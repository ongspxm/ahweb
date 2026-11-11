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
)

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error { _, err := exec.LookPath("cursor-agent"); return err }

type conn struct {
	cmd          *exec.Cmd
	stderr       bytes.Buffer
	events       []harness.Event
	cursorID     string
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
	cacheMu   sync.Mutex
	cache     = make(map[string]map[string]*conn)
	cursorIDs = make(map[string]map[string]string)
)

func (*Adapter) GetModels() (map[string]harness.Model, error) {
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
	for _, line := range strings.Split(string(output), "\n") {
		id, _, ok := strings.Cut(strings.TrimSpace(line), " - ")
		if !ok || id == "" {
			continue
		}
		id = "cursor/" + id
		models[id] = harness.Model{Harness: "cursor", ID: id}
	}
	if len(models) == 0 {
		return nil, errors.New("Cursor Agent returned no models")
	}
	return models, nil
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
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		cursorID := entry.Name()
		path := filepath.Join(root, cursorID, cursorID+".jsonl")
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		info, err := file.Stat()
		if err != nil {
			_ = file.Close()
			continue
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
			continue
		}

		// A session created in this run has a local ID; do not list its chat twice.
		sessID := cursorID
		cacheMu.Lock()
		for cachedSessID, cachedCursorID := range cursorIDs[realFolder] {
			if cachedCursorID == cursorID || sessionUUID(cachedSessID) == cursorID {
				sessID = cachedSessID
				break
			}
		}
		cacheMu.Unlock()
		lite := harness.SessionLite{ID: sessID, RealPath: realFolder, Harness: "cursor", FirstUserMsg: first, LastTimestampMs: info.ModTime().UnixMilli()}
		sessions = append(sessions, lite)
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
	if cursorIDs[s.RealPath] == nil {
		cursorIDs[s.RealPath] = make(map[string]string)
	}
	if cache[s.RealPath][s.ID] == nil {
		cursorID := cursorIDs[s.RealPath][s.ID]
		if cursorID == "" {
			cursorID = sessionUUID(s.ID)
			if _, err := os.Stat(transcriptPath(s.RealPath, cursorID)); err != nil {
				cursorID = ""
			}
		}
		cache[s.RealPath][s.ID] = &conn{events: append([]harness.Event(nil), s.Events...), cursorID: cursorID}
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
	if cmd != nil && cmd.Process != nil {
		c.isStopped = true
		_ = cmd.Process.Kill()
	}
	c.mu.Unlock()
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
	c, err := a.conn(s)
	if err != nil {
		return err
	}
	s.Mu.Lock()
	prompt, userPrompt, images, model := s.SendingMsg, s.SendingMsg, append([]string(nil), s.SendingImages...), s.PromptModel
	s.Mu.Unlock()
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
	if c.cursorID != "" {
		args = append(args, "--resume", c.cursorID)
	}
	if model != nil && model.ID != "" {
		args = append(args, "--model", strings.TrimPrefix(model.ID, "cursor/"))
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
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start Cursor Agent: %w", err)
	}
	keepImages = true
	done := make(chan struct{})
	c.cmd, c.isStopped, c.processDone = cmd, false, done
	c.events = append(c.events, harness.Event{Type: harness.EventUser, Content: userPrompt, Images: images, TimestampMs: time.Now().UnixMilli()})
	s.Mu.Lock()
	s.BusyCli = true
	s.Mu.Unlock()

	// process the stdout form the cmd call
	go func() {
		defer func() {
			for _, path := range imageFiles {
				_ = os.Remove(path)
			}
		}()
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
				c.events = append(c.events, event)
				c.mu.Unlock()
			}
			switch kind {
			case "system":
				if subtype, _ := msg["subtype"].(string); subtype == "init" {
					if cursorID, _ := msg["session_id"].(string); cursorID != "" {
						c.mu.Lock()
						c.cursorID = cursorID
						c.mu.Unlock()
						cacheMu.Lock()
						if cursorIDs[s.RealPath] == nil {
							cursorIDs[s.RealPath] = make(map[string]string)
						}
						cursorIDs[s.RealPath][s.ID] = cursorID
						cacheMu.Unlock()
					}
				}
			case "user":
				continue
			case "assistant":
				message, _ := msg["message"].(map[string]any)
				for _, part := range contentParts(message["content"]) {
					if text, _ := part["text"].(string); text != "" {
						_add(harness.Event{Type: harness.EventAssistant, Content: text})
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
		s.Mu.Lock()
		s.BusyCli = false
		s.Mu.Unlock()
		close(done)
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
	cursorID := c.cursorID

	// full session reading if cursor sess exist
	if fullHistory && cursorID != "" {
		file, err := os.Open(transcriptPath(s.RealPath, cursorID))
		if err == nil {
			if info, err := file.Stat(); err == nil {
				stamp := info.ModTime().UnixMilli()
				var events []harness.Event
				scanner := bufio.NewScanner(file)
				scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
				for scanner.Scan() {
					var row struct {
						Role    string `json:"role"`
						Message struct {
							Content any `json:"content"`
						} `json:"message"`
					}
					if json.Unmarshal(scanner.Bytes(), &row) != nil {
						continue
					}
					typ := harness.EventAssistant
					if row.Role == "user" {
						typ = harness.EventUser
					} else if row.Role != "assistant" {
						continue
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
							events = append(events, harness.Event{Type: typ, Content: text, TimestampMs: stamp})
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
	for _, event := range events {
		if event.TimestampMs > s.LastTimestampMs {
			s.LastTimestampMs = event.TimestampMs
		}
		if event.Type == harness.EventUser && s.FirstUserMsg == "" {
			s.FirstUserMsg = event.Content
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
	defer c.mu.Unlock()
	if c.cmd == nil {
		return nil
	}
	if err := c.cmd.Process.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("could not interrupt Cursor Agent: %w", err)
	}
	c.isStopped = true
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

func transcriptPath(realPath, cursorID string) string {
	// Mirror Cursor's project folder name: replace separators and dots, then collapse dashes.
	key := strings.NewReplacer("\\", "-", "/", "-", ".", "-").Replace(realPath)
	for strings.Contains(key, "--") {
		key = strings.ReplaceAll(key, "--", "-")
	}
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".cursor", "projects", strings.Trim(key, "-"), "agent-transcripts")
	if cursorID == "" {
		return root
	}
	return filepath.Join(root, cursorID, cursorID+".jsonl")
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
