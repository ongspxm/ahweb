package cursor

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
	"github.com/ongspxm/ahweb/internal/processgroup"
)

// === adapter admin fns

type Adapter struct {
	harness.DefaultAdapter

	// cursor have modelID and modelName and for some reason
	// it only returns the modelName in the stream-json
	// but we need the modelID to submit msgs
	// so this maps both modelID and modelName to modelID
	modelIDsMu sync.RWMutex
	modelIDs   map[string]string
}

func (*Adapter) Ready() error { _, err := exec.LookPath("cursor-agent"); return err }

type conn struct {
	cmd          *exec.Cmd
	stderr       bytes.Buffer
	events       []harness.Event
	harnessID    string
	parentID     string
	isStopped    bool
	isClosing    bool
	processDone  chan struct{}
	inputTokens  int64
	outputTokens int64
	cacheRead    int64
	cacheWrite   int64
	cost         float64
	mu           sync.Mutex
}

var (
	cacheMu           sync.Mutex
	cache             = make(map[string]map[string]*conn)
	harnessIDToSessID = make(map[string]map[string]string)
	sessIDToHarnessID = make(map[string]map[string]string)
)

func (a *Adapter) GetModels() (map[string]harness.Model, error) {
	cmd := exec.Command("cursor-agent", "--list-models")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("could not list Cursor models: %s", strings.TrimSpace(stderr.String()))
		}
		return nil, fmt.Errorf("could not list Cursor models: %w", err)
	}
	models := make(map[string]harness.Model)
	modelIDs := make(map[string]string)
	ambiguousNames := make(map[string]bool)
	_addModelName := func(name, id string) {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" || ambiguousNames[name] {
			return
		}
		if previous := modelIDs[name]; previous != "" && previous != id {
			delete(modelIDs, name)
			ambiguousNames[name] = true
			return
		}
		modelIDs[name] = id
	}
	for _, line := range strings.Split(string(output), "\n") {
		id, name, ok := strings.Cut(strings.TrimSpace(line), " - ")
		if !ok || id == "" {
			continue
		}
		models[id] = harness.Model{Harness: "cursor", ID: id}
		_addModelName(id, id)
		_addModelName(name, id)
	}
	if len(models) == 0 {
		return nil, errors.New("Cursor Agent returned no models")
	}
	a.modelIDsMu.Lock()
	a.modelIDs = modelIDs
	a.modelIDsMu.Unlock()
	return models, nil
}

func (a *Adapter) _modelID(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	a.modelIDsMu.RLock()
	id := a.modelIDs[model]
	a.modelIDsMu.RUnlock()
	return id
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	root := transcriptPath(realFolder, "")
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []harness.SessionLite{}, nil
		}
		return nil, fmt.Errorf("could not read Cursor sessions: %w", err)
	}
	var sessions []harness.SessionLite
	_read := func(harnessID, path, parentID string) string {
		file, err := os.Open(path)
		if err != nil {
			return ""
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return ""
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		first, hasText := "", false
		for scanner.Scan() {
			var row struct {
				Role    string `json:"role"`
				Message struct {
					Content any `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(scanner.Bytes(), &row) != nil || (row.Role != "user" && row.Role != "assistant") {
				continue
			}
			for _, part := range contentParts(row.Message.Content) {
				text, _ := part["text"].(string)
				if text == "" {
					continue
				}
				hasText = true
				if row.Role == "user" {
					if start := strings.Index(text, "<user_query>"); start >= 0 {
						text = text[start+len("<user_query>"):]
						if end := strings.Index(text, "</user_query>"); end >= 0 {
							text = text[:end]
						}
					}
					first = strings.TrimSpace(text)
					break
				}
			}
			if row.Role == "user" && hasText {
				break
			}
		}
		scanErr := scanner.Err()
		_ = file.Close()
		if scanErr != nil || !hasText {
			return ""
		}

		// A session created in this run has a local ID; do not list its chat twice.
		sessID := harnessID
		cacheMu.Lock()
		if mappedID := harnessIDToSessID[realFolder][harnessID]; mappedID != "" {
			sessID = mappedID
		}
		cacheMu.Unlock()
		sessions = append(sessions, harness.SessionLite{ID: sessID, RealPath: realFolder, Harness: "cursor", FirstUserMsg: first, LastTimestampMs: info.ModTime().UnixMilli(), ParentSessID: parentID, ReadOnly: parentID != ""})
		return sessID
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		harnessID := entry.Name()
		path := filepath.Join(root, harnessID, harnessID+".jsonl")
		parentID := _read(harnessID, path, "")
		if parentID == "" {
			continue
		}
		children, err := os.ReadDir(filepath.Join(root, harnessID, "subagents"))
		if err != nil {
			continue
		}
		for _, child := range children {
			if child.IsDir() || filepath.Ext(child.Name()) != ".jsonl" {
				continue
			}
			childID := strings.TrimSuffix(child.Name(), ".jsonl")
			_read(childID, filepath.Join(root, harnessID, "subagents", child.Name()), parentID)
		}
	}
	return sessions, nil
}

// === session admin fns

func (a *Adapter) Connect(s *harness.Session, _ bool) error {
	if _, err := exec.LookPath("cursor-agent"); err != nil {
		return fmt.Errorf("Cursor Agent CLI is not installed: %w", err)
	}
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cache[s.RealPath] == nil {
		cache[s.RealPath] = make(map[string]*conn)
	}
	if harnessIDToSessID[s.RealPath] == nil {
		harnessIDToSessID[s.RealPath] = make(map[string]string)
	}
	if sessIDToHarnessID[s.RealPath] == nil {
		sessIDToHarnessID[s.RealPath] = make(map[string]string)
	}
	if cache[s.RealPath][s.ID] == nil {
		harnessID := sessionUUID(s.ID)
		parentID := sessionUUID(s.ParentSessID)
		path := transcriptPath(s.RealPath, harnessID)
		if s.ParentSessID != "" {
			if mappedID := sessIDToHarnessID[s.RealPath][s.ParentSessID]; mappedID != "" {
				parentID = mappedID
			}
			path = childTranscriptPath(s.RealPath, parentID, harnessID)
		}
		if _, err := os.Stat(path); err != nil {
			harnessID = ""
		}
		cache[s.RealPath][s.ID] = &conn{events: append([]harness.Event(nil), s.Events...), harnessID: harnessID, parentID: parentID}
		if harnessID != "" {
			harnessIDToSessID[s.RealPath][harnessID] = s.ID
			sessIDToHarnessID[s.RealPath][s.ID] = harnessID
		}
	}
	return nil
}

func (*Adapter) CleanUp(s *harness.Session) error {
	cacheMu.Lock()
	c := cache[s.RealPath][s.ID]
	cacheMu.Unlock()
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.isClosing = true
	cmd, done := c.cmd, c.processDone
	if cmd != nil {
		c.isStopped = true
	}
	c.mu.Unlock()
	_ = processgroup.Kill(cmd)
	if done != nil {
		<-done
	}
	cacheMu.Lock()
	if cache[s.RealPath][s.ID] == c {
		delete(cache[s.RealPath], s.ID)
		if len(cache[s.RealPath]) == 0 {
			delete(cache, s.RealPath)
		}
	}
	cacheMu.Unlock()
	return nil
}

// === session fns

func (a *Adapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	prompt, userPrompt, images, model := s.SendingMsg, s.SendingMsg, append([]string(nil), s.SendingImages...), s.PromptModel
	s.BusyCli = true
	s.Mu.Unlock()
	c, err := a.conn(s)
	if err != nil {
		return err
	}
	liveModel := ""
	if model != nil {
		liveModel = model.ID
	}
	var imageFiles []string
	keepImages := false
	defer func() {
		if !keepImages {
			for _, path := range imageFiles {
				_ = os.Remove(path)
			}
		}
	}()
	for _, uri := range images {
		image, err := harness.ParseImage(uri)
		if err != nil {
			return err
		}
		data, err := base64.StdEncoding.DecodeString(image.Data)
		if err != nil {
			return fmt.Errorf("invalid Cursor image data: %w", err)
		}
		file, err := os.CreateTemp("", "cursor-image-*")
		if err != nil {
			return fmt.Errorf("could not create Cursor image: %w", err)
		}
		imageFiles = append(imageFiles, file.Name())
		if _, err = file.Write(data); err == nil {
			err = file.Close()
		} else {
			_ = file.Close()
		}
		if err != nil {
			return fmt.Errorf("could not write Cursor image: %w", err)
		}
		prompt += fmt.Sprintf("\n\nImage %d: %s", len(imageFiles), file.Name())
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil || c.isClosing {
		return errors.New("Cursor session is busy")
	}
	args := []string{"-p", "--output-format", "stream-json", "--force"}
	if c.harnessID != "" {
		args = append(args, "--resume", c.harnessID)
	}
	if model != nil && model.ID != "" {
		args = append(args, "--model", model.ID)
	}
	args = append(args, prompt)
	cmd := exec.Command("cursor-agent", args...)
	cmd.Dir = s.RealPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not open Cursor output: %w", err)
	}
	c.stderr.Reset()
	cmd.Stderr = &c.stderr
	if err := processgroup.Start(cmd); err != nil {
		return fmt.Errorf("could not start Cursor Agent: %w", err)
	}
	keepImages = true
	done := make(chan struct{})
	c.cmd, c.isStopped, c.processDone = cmd, false, done
	c.events = append(c.events, harness.Event{Type: harness.EventUser, Content: userPrompt, Images: images, TimestampMs: time.Now().UnixMilli()})

	// process the stdout form the cmd call
	go func() {
		defer close(done)
		defer func() {
			for _, path := range imageFiles {
				_ = os.Remove(path)
			}
		}()
		currentModel := liveModel
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			var msg map[string]any
			if json.Unmarshal(scanner.Bytes(), &msg) != nil {
				continue
			}
			kind, _ := msg["type"].(string)
			_add := func(event harness.Event) {
				event.TimestampMs = time.Now().UnixMilli()
				c.mu.Lock()
				if (event.Type == harness.EventAssistant || event.Type == harness.EventThinking) && event.Model == "" {
					event.Model = currentModel
				}
				c.events = append(c.events, event)
				c.mu.Unlock()
			}
			_toolArgs := func(args map[string]any) string {
				for _, key := range []string{"command", "pattern", "path", "glob_pattern", "globPattern", "search_term", "description", "prompt"} {
					if v, _ := args[key].(string); strings.TrimSpace(v) != "" {
						return strings.TrimSpace(v)
					}
				}
				if raw, err := json.Marshal(args); err == nil {
					return string(raw)
				}
				return ""
			}
			switch kind {
			case "system":
				if subtype, _ := msg["subtype"].(string); subtype == "init" {
					model, _ := msg["model"].(string)
					harnessID, _ := msg["session_id"].(string)
					modelID := a._modelID(model)
					if modelID != "" {
						currentModel = modelID
					}
					if harnessID != "" {
						c.mu.Lock()
						c.harnessID = harnessID
						c.mu.Unlock()
						cacheMu.Lock()
						harnessIDToSessID[s.RealPath][harnessID] = s.ID
						sessIDToHarnessID[s.RealPath][s.ID] = harnessID
						cacheMu.Unlock()
					}
				}
			case "user":
				continue
			case "assistant":
				model, _ := msg["model"].(string)
				reasoning, _ := msg["effort"].(string)
				if model == "" {
					if message, ok := msg["message"].(map[string]any); ok {
						model, _ = message["model"].(string)
						if reasoning == "" {
							reasoning, _ = message["effort"].(string)
						}
					}
				}
				model = a._modelID(model)
				message, _ := msg["message"].(map[string]any)
				for _, part := range contentParts(message["content"]) {
					if text, _ := part["text"].(string); text != "" {
						_add(harness.Event{Type: harness.EventAssistant, Content: text, Model: model, Reasoning: reasoning})
					}
				}
			case "tool_call":
				// Parse tool events here; this stream block has no other caller.
				subtype, _ := msg["subtype"].(string)
				event := harness.Event{}
				event.ToolCallID, _ = msg["call_id"].(string)
				tool, _ := msg["tool_call"].(map[string]any)
				name, _ := tool["name"].(string)
				if name == "" {
					name, _ = tool["tool_name"].(string)
				}
				var call map[string]any
				for key, value := range tool {
					candidate, ok := value.(map[string]any)
					if !ok {
						continue
					}
					call = candidate
					if name == "" {
						name, _ = candidate["name"].(string)
						if name == "" {
							name = strings.TrimSuffix(key, "ToolCall")
						}
					}
					break
				}
				event.ToolName = name
				if subtype == "started" {
					event.Type = harness.EventToolStart
					if args, _ := call["args"].(map[string]any); args != nil {
						event.Content = _toolArgs(args)
					}
					_add(event)
					continue
				}
				if subtype != "completed" {
					continue
				}
				event.Type = harness.EventToolEnd
				result, _ := call["result"].(map[string]any)
				if success, ok := result["success"].(map[string]any); ok {
					event.Content, _ = success["content"].(string)
					if event.Content == "" {
						event.Content, _ = success["stdout"].(string)
					}
				} else if result != nil {
					event.ToolFailed = true
					if raw, err := json.Marshal(result); err == nil {
						event.Content = string(raw)
					}
				}
				_add(event)
			case "result":
				var result struct {
					Subtype string `json:"subtype"`
					Content string `json:"result"`
					IsError bool   `json:"is_error"`
					Usage   struct {
						InputTokens      int64 `json:"inputTokens"`
						OutputTokens     int64 `json:"outputTokens"`
						CacheReadTokens  int64 `json:"cacheReadTokens"`
						CacheWriteTokens int64 `json:"cacheWriteTokens"`
					} `json:"usage"`
					Cost *float64 `json:"cost"`
				}
				if json.Unmarshal(scanner.Bytes(), &result) == nil {
					c.mu.Lock()
					c.inputTokens += result.Usage.InputTokens
					c.outputTokens += result.Usage.OutputTokens
					c.cacheRead += result.Usage.CacheReadTokens
					c.cacheWrite += result.Usage.CacheWriteTokens
					if result.Cost != nil {
						c.cost += *result.Cost
					}
					c.mu.Unlock()
					if strings.Contains(result.Subtype, "error") || result.IsError {
						_add(harness.Event{Type: harness.EventError, Content: result.Content})
					}
				}
			}
		}
		scanErr := scanner.Err()
		waitErr := cmd.Wait()
		c.mu.Lock()
		stderr := strings.TrimSpace(c.stderr.String())
		isStopped := c.isStopped
		c.isStopped = false
		if scanErr != nil {
			c.events = append(c.events, harness.Event{Type: harness.EventError, Content: "could not read Cursor output: " + scanErr.Error(), TimestampMs: time.Now().UnixMilli()})
		} else if waitErr != nil && !isStopped {
			if stderr == "" {
				stderr = waitErr.Error()
			}
			c.events = append(c.events, harness.Event{Type: harness.EventError, Content: "Cursor Agent failed: " + stderr, TimestampMs: time.Now().UnixMilli()})
		}
		c.cmd = nil
		c.mu.Unlock()
	}()
	return nil
}

func (a *Adapter) Poll(s *harness.Session) error {
	c, err := a.conn(s)
	if err != nil {
		return err
	}
	s.Mu.Lock()
	fullHistory := s.DoFullHistory
	s.Mu.Unlock()
	c.mu.Lock()
	harnessID := c.harnessID
	parentID := c.parentID

	// full session reading if cursor sess exist
	if fullHistory && harnessID != "" {
		path := transcriptPath(s.RealPath, harnessID)
		if s.ParentSessID != "" {
			path = childTranscriptPath(s.RealPath, parentID, harnessID)
		}
		file, err := os.Open(path)
		if err == nil {
			if info, err := file.Stat(); err == nil {
				stamp := info.ModTime().UnixMilli()
				var events []harness.Event
				currentModel, currentReasoning := "", ""
				scanner := bufio.NewScanner(file)
				scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
				for scanner.Scan() {
					var row struct {
						Type    string         `json:"type"`
						Role    string         `json:"role"`
						Model   string         `json:"model"`
						Effort  string         `json:"effort"`
						Data    map[string]any `json:"data"`
						Message struct {
							Content any    `json:"content"`
							Model   string `json:"model"`
							Effort  string `json:"effort"`
						} `json:"message"`
					}
					if json.Unmarshal(scanner.Bytes(), &row) != nil {
						continue
					}
					model := row.Model
					if model == "" {
						model, _ = row.Data["model"].(string)
					}
					if model == "" {
						model = row.Message.Model
					}
					model = a._modelID(model)
					reasoning := row.Effort
					if reasoning == "" {
						reasoning, _ = row.Data["effort"].(string)
					}
					if reasoning == "" {
						reasoning = row.Message.Effort
					}
					if row.Type == "system" || row.Role == "system" {
						if model != "" {
							currentModel = model
						}
						if reasoning != "" {
							currentReasoning = reasoning
						}
						continue
					}
					typ := harness.EventAssistant
					if row.Role == "user" {
						typ = harness.EventUser
					} else if row.Role != "assistant" {
						continue
					}
					if model == "" {
						model = currentModel
					}
					if reasoning == "" {
						reasoning = currentReasoning
					}
					if typ != harness.EventAssistant {
						model, reasoning = "", ""
					} else {
						if model != "" {
							currentModel = model
						}
						if reasoning != "" {
							currentReasoning = reasoning
						}
					}

					// Saved tool calls have name and input but no result or call ID, so replay only the start.
					for _, part := range contentParts(row.Message.Content) {
						if part["type"] == "tool_use" {
							name, _ := part["name"].(string)
							input, err := json.Marshal(part["input"])
							if err == nil {
								events = append(events, harness.Event{Type: harness.EventToolStart, ToolName: name, Content: string(input), TimestampMs: stamp})
							}
							continue
						}
						if text, _ := part["text"].(string); text != "" {
							if typ == harness.EventUser {
								if start := strings.Index(text, "<user_query>"); start >= 0 {
									text = text[start+len("<user_query>"):]
									if end := strings.Index(text, "</user_query>"); end >= 0 {
										text = text[:end]
									}
								}
								text = strings.TrimSpace(text)
							}
							events = append(events, harness.Event{Type: typ, Content: text, TimestampMs: stamp, Model: model, Reasoning: reasoning})
						}
					}
				}
				if scanner.Err() == nil && len(events) > 0 {
					c.events = events
				}
			}
			_ = file.Close()
		}
	}

	events := append([]harness.Event(nil), c.events...)
	inputTokens, outputTokens, cacheRead, cacheWrite, cost := c.inputTokens, c.outputTokens, c.cacheRead, c.cacheWrite, c.cost
	busy := c.cmd != nil
	c.mu.Unlock()
	s.Mu.Lock()
	s.Events, s.BusyCli, s.DoFullHistory = events, busy, false
	s.FullInputToken, s.FullOutputToken = inputTokens, outputTokens
	s.CacheReadToken, s.CacheWriteToken, s.FullCost = cacheRead, cacheWrite, cost
	if fullHistory {
		s.FirstUserMsg = ""
	}
	firstUserFound := s.FirstUserMsg != ""
	for _, event := range events {
		if event.TimestampMs > s.LastTimestampMs {
			s.LastTimestampMs = event.TimestampMs
		}
		if event.Type == harness.EventUser && !firstUserFound {
			s.FirstUserMsg, firstUserFound = event.Content, true
		}
	}
	s.Mu.Unlock()
	return nil
}

func (a *Adapter) Stop(s *harness.Session) error {
	c, err := a.conn(s)
	if err != nil {
		return nil
	}
	c.mu.Lock()
	cmd, done := c.cmd, c.processDone
	if cmd == nil {
		c.mu.Unlock()
		return nil
	}
	c.isStopped = true
	c.mu.Unlock()
	if err := processgroup.Stop(cmd, done, time.Second); err != nil {
		return fmt.Errorf("could not stop Cursor Agent: %w", err)
	}
	return nil
}

// === utils fns

func (*Adapter) conn(s *harness.Session) (*conn, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c := cache[s.RealPath][s.ID]; c != nil {
		return c, nil
	}
	return nil, errors.New("Cursor session is not connected")
}

func sessionUUID(sessID string) string {
	if strings.Contains(sessID, "-") {
		return sessID
	}
	decoded, err := hex.DecodeString(sessID)
	if err != nil || len(decoded) != 16 {
		return sessID
	}
	decoded[6] = decoded[6]&0x0f | 0x40
	decoded[8] = decoded[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", decoded[:4], decoded[4:6], decoded[6:8], decoded[8:10], decoded[10:])
}

func transcriptPath(realPath, harnessID string) string {
	// Mirror Cursor's project folder name: replace separators and dots, then collapse dashes.
	key := strings.NewReplacer("\\", "-", "/", "-", ".", "-").Replace(realPath)
	for strings.Contains(key, "--") {
		key = strings.ReplaceAll(key, "--", "-")
	}
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".cursor", "projects", strings.Trim(key, "-"), "agent-transcripts")
	if harnessID == "" {
		return root
	}
	return filepath.Join(root, harnessID, harnessID+".jsonl")
}

func childTranscriptPath(realPath, parentID, childID string) string {
	return filepath.Join(transcriptPath(realPath, ""), parentID, "subagents", childID+".jsonl")
}

func contentParts(content any) []map[string]any {
	switch value := content.(type) {
	case string:
		return []map[string]any{{"type": "text", "text": value}}
	case []any:
		var parts []map[string]any
		for _, item := range value {
			if part, ok := item.(map[string]any); ok {
				parts = append(parts, part)
			}
		}
		return parts
	}
	return nil
}
