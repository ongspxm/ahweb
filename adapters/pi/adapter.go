package pi

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type connStruct struct {
	Mu      sync.Mutex
	Cmd     *exec.Cmd
	Stdin   io.WriteCloser
	Stdout  io.ReadCloser
	Decoder *json.Decoder
	Cursor  string
}

var (
	cacheMu    sync.Mutex
	cacheConns = make(map[string]map[string]*connStruct)
)

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error { _, err := exec.LookPath("pi"); return err }

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	cmd := exec.Command("pi", "--mode", "rpc", "--no-session")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Pi input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Pi output: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start Pi: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()
	timer := time.AfterFunc(30*time.Second, func() { _ = cmd.Process.Kill() })
	defer timer.Stop()
	if _, err := io.WriteString(stdin, `{"id":"models","type":"get_available_models"}`+"\n"); err != nil {
		return nil, fmt.Errorf("could not ask Pi for models: %w", err)
	}
	var response struct {
		Type    string `json:"type"`
		Command string `json:"command"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Models []struct {
				Provider         string             `json:"provider"`
				ID               string             `json:"id"`
				Reasoning        bool               `json:"reasoning"`
				ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
			} `json:"models"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(stdout)
	for decoder.Decode(&response) == nil {
		if response.Type != "response" || response.Command != "get_available_models" {
			continue
		}
		if !response.Success {
			return nil, fmt.Errorf("Pi refused model list: %s", response.Error)
		}
		models := make(map[string]harness.Model, len(response.Data.Models))
		for _, model := range response.Data.Models {
			modelID := model.Provider + "/" + model.ID
			entry := harness.Model{Harness: "pi", ID: modelID}
			if model.Reasoning {
				for _, level := range []string{"minimal", "low", "medium", "high", "xhigh", "max"} {
					if effort, ok := model.ThinkingLevelMap[level]; ok && effort != nil {
						entry.Reasoning = append(entry.Reasoning, level)
					}
				}
				for _, level := range []string{"medium", "high", "xhigh", "max", "low", "minimal"} {
					for i, reasoning := range entry.Reasoning {
						if reasoning == level {
							entry.Reasoning = append([]string{reasoning}, append(entry.Reasoning[:i], entry.Reasoning[i+1:]...)...)
							break
						}
					}
					if len(entry.Reasoning) > 0 && entry.Reasoning[0] == level {
						break
					}
				}
			}
			models[modelID] = entry
		}
		return models, nil
	}
	return nil, errors.New("Pi closed before sending model list")
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find Pi home: %w", err)
	}
	encoded := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(strings.TrimLeft(realFolder, "/\\"))
	dir := filepath.Join(home, ".pi", "agent", "sessions", "--"+encoded+"--")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not read Pi sessions: %w", err)
	}
	sessions := make([]harness.SessionLite, 0)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		file, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		var header struct {
			Type          string `json:"type"`
			ID            string `json:"id"`
			CWD           string `json:"cwd"`
			ParentSession string `json:"parentSession"`
		}
		if scanner.Scan() && json.Unmarshal(scanner.Bytes(), &header) == nil && header.Type == "session" && header.ID != "" {
			s := harness.SessionLite{ID: header.ID, RealPath: header.CWD, Harness: "pi"}
			if parent, err := os.Open(header.ParentSession); err == nil {
				parentScanner := bufio.NewScanner(parent)
				if parentScanner.Scan() {
					var parentHeader struct {
						Type string `json:"type"`
						ID   string `json:"id"`
					}
					if json.Unmarshal(parentScanner.Bytes(), &parentHeader) == nil && parentHeader.Type == "session" {
						s.ParentSessID = parentHeader.ID
					}
				}
				_ = parent.Close()
			}
			var firstMessage string
			gotFirstMessage := false
			for scanner.Scan() {
				var entry struct {
					Type      string `json:"type"`
					Name      string `json:"name"`
					Timestamp string `json:"timestamp"`
					Message   struct {
						Role    string          `json:"role"`
						Content json.RawMessage `json:"content"`
					} `json:"message"`
				}
				if json.Unmarshal(scanner.Bytes(), &entry) != nil {
					continue
				}
				if entry.Type == "session_info" {
					s.Name = entry.Name
				}
				if entry.Type == "message" && entry.Message.Role == "user" && !gotFirstMessage {
					gotFirstMessage = true
					if json.Unmarshal(entry.Message.Content, &firstMessage) != nil {
						var parts []struct {
							Type string `json:"type"`
							Text string `json:"text"`
						}
						if json.Unmarshal(entry.Message.Content, &parts) == nil {
							for _, part := range parts {
								if part.Type == "text" {
									firstMessage += part.Text
								}
							}
						}
					}
				}
				if timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil && timestamp.UnixMilli() > s.LastTimestampMs {
					s.LastTimestampMs = timestamp.UnixMilli()
				}
			}
			s.FirstUserMsg = firstMessage
			sessions = append(sessions, s)
		}
		_ = file.Close()
	}
	return sessions, nil
}

// === session admin fns

func (*Adapter) CleanUp(s *harness.Session) error {
	cacheMu.Lock()
	conn := cacheConns[s.RealPath][s.ID]
	cacheMu.Unlock()
	if conn == nil {
		return nil
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.Cmd != nil {
		_ = conn.Stdin.Close()
		if conn.Cmd.ProcessState == nil {
			_ = conn.Cmd.Process.Kill()
		}
		_ = conn.Cmd.Wait()
		conn.Cmd, conn.Stdin, conn.Stdout, conn.Decoder = nil, nil, nil, nil
		conn.Cursor = ""
	}
	cacheMu.Lock()
	if sessions := cacheConns[s.RealPath]; sessions[s.ID] == conn {
		delete(sessions, s.ID)
		if len(sessions) == 0 {
			delete(cacheConns, s.RealPath)
		}
	}
	cacheMu.Unlock()
	return nil
}

func (a *Adapter) Connect(s *harness.Session, force bool) error {
	folder := s.RealPath
	cacheMu.Lock()
	if cacheConns[folder] == nil {
		cacheConns[folder] = make(map[string]*connStruct)
	}
	conn := cacheConns[folder][s.ID]
	if conn == nil {
		conn = &connStruct{}
		cacheConns[folder][s.ID] = conn
	}
	cacheMu.Unlock()
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.Cmd != nil {
		if !force && conn.Cmd.ProcessState == nil {
			return nil
		}
		conn.Mu.Unlock()
		_ = a.CleanUp(s)
		conn.Mu.Lock()
		cacheMu.Lock()
		if cacheConns[folder] == nil {
			cacheConns[folder] = make(map[string]*connStruct)
		}
		cacheConns[folder][s.ID] = conn
		cacheMu.Unlock()
	}
	cmd := exec.Command("pi", "--mode", "rpc", "--session-id", s.ID)
	cmd.Dir = folder
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("could not open Pi input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return fmt.Errorf("could not open Pi output: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("could not start Pi: %w", err)
	}
	conn.Cmd, conn.Stdin, conn.Stdout, conn.Decoder = cmd, stdin, stdout, json.NewDecoder(stdout)

	// if new connection created, i want to force a full history for consistency
	s.Mu.Lock()
	s.DoFullHistory = true
	defer s.Mu.Unlock()
	return nil
}

func (*Adapter) CanDelete() bool { return true }

func (a *Adapter) Delete(s *harness.Session) error {
	if s.GetIsBusy() {
		return errors.New("cannot delete a busy Pi session")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	encoded := strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(strings.TrimLeft(s.RealPath, "/\\"))
	dir := filepath.Join(home, ".pi", "agent", "sessions", "--"+encoded+"--")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var sessionPath string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		var header struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			CWD  string `json:"cwd"`
		}
		err = json.NewDecoder(bufio.NewReader(file)).Decode(&header)
		_ = file.Close()
		if err == nil && header.Type == "session" && header.ID == s.ID && header.CWD == s.RealPath {
			sessionPath = path
			break
		}
	}
	if sessionPath == "" {
		return fmt.Errorf("Pi session %q was not found", s.ID)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" || filepath.Join(dir, entry.Name()) == sessionPath {
			continue
		}
		file, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var header struct {
			ParentSession string `json:"parentSession"`
		}
		err = json.NewDecoder(bufio.NewReader(file)).Decode(&header)
		_ = file.Close()
		if err == nil && header.ParentSession != "" {
			parent, _ := filepath.Abs(header.ParentSession)
			current, _ := filepath.Abs(sessionPath)
			if filepath.Clean(parent) == filepath.Clean(current) {
				return errors.New("cannot delete a Pi session with child sessions")
			}
		}
	}
	if err := a.CleanUp(s); err != nil {
		return err
	}
	return os.Remove(sessionPath)
}

// === session fns

func (a *Adapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	model, reasoning, prompt := s.PromptModel, s.PromptReasoning, s.SendingMsg
	images := append([]string(nil), s.SendingImages...)
	s.Mu.Unlock()
	cacheMu.Lock()
	conn := cacheConns[s.RealPath][s.ID]
	cacheMu.Unlock()
	if conn == nil {
		return errors.New("Pi session is not running")
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.Cmd == nil || conn.Cmd.ProcessState != nil {
		return errors.New("Pi session is not running")
	}

	request := func(command map[string]any) error {
		id := fmt.Sprintf("prompt-%d", time.Now().UnixNano())
		command["id"] = id
		if err := json.NewEncoder(conn.Stdin).Encode(command); err != nil {
			return fmt.Errorf("could not send Pi command: %w", err)
		}
		var response struct {
			Type    string `json:"type"`
			ID      string `json:"id"`
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		for {
			if err := conn.Decoder.Decode(&response); err != nil {
				return fmt.Errorf("could not read Pi response: %w", err)
			}
			if response.Type != "response" || response.ID != id {
				continue
			}
			if !response.Success {
				return fmt.Errorf("Pi refused command: %s", response.Error)
			}
			return nil
		}
	}
	if model != nil {
		provider, modelID, ok := strings.Cut(model.ID, "/")
		if !ok || provider == "" || modelID == "" {
			return fmt.Errorf("invalid Pi model ID: %q", model.ID)
		}
		if err := request(map[string]any{"type": "set_model", "provider": provider, "modelId": modelID}); err != nil {
			return err
		}
		if reasoning == "" {
			if len(model.Reasoning) > 0 {
				reasoning = model.Reasoning[0]
			}
		}
		if reasoning != "" {
			if err := request(map[string]any{"type": "set_thinking_level", "level": reasoning}); err != nil {
				return err
			}
		}
	}
	command := map[string]any{"type": "prompt", "message": prompt}
	if len(images) > 0 {
		parts := make([]harness.Image, 0, len(images))
		for _, uri := range images {
			part, err := harness.ParseImage(uri)
			if err != nil {
				return err
			}
			parts = append(parts, part)
		}
		command["images"] = parts
	}
	if err := request(command); err != nil {
		return err
	}
	// Mark the session busy so the web UI will poll.
	s.Mu.Lock()
	s.BusyCli = true
	s.Mu.Unlock()
	return nil
}

func (a *Adapter) Stop(s *harness.Session) error {
	conn, err := a.conn(s)
	if err != nil {
		return nil
	}

	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.Cmd == nil || conn.Cmd.ProcessState != nil {
		return nil
	}
	id := fmt.Sprintf("stop-%d", time.Now().UnixNano())
	if err := json.NewEncoder(conn.Stdin).Encode(map[string]string{"id": id, "type": "abort"}); err != nil {
		return fmt.Errorf("could not request Pi stop: %w", err)
	}
	var response struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
	}
	for {
		if err := conn.Decoder.Decode(&response); err != nil {
			return fmt.Errorf("could not read Pi stop response: %w", err)
		}
		if response.Type != "response" || response.ID != id {
			continue
		}
		if !response.Success {
			return fmt.Errorf("Pi refused stop: %s", response.Error)
		}
		return nil
	}
}

func (a *Adapter) Poll(s *harness.Session) error {
	conn, err := a.conn(s)
	if err != nil {
		return err
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.Cmd == nil || conn.Cmd.ProcessState != nil {
		return errors.New("Pi session is not running")
	}

	s.Mu.Lock()
	full := s.DoFullHistory
	s.Mu.Unlock()

	id := fmt.Sprintf("poll-%d", time.Now().UnixNano())
	command := map[string]any{"id": id, "type": "get_entries"}
	if !full && conn.Cursor != "" {
		command["since"] = conn.Cursor
	}
	if err := json.NewEncoder(conn.Stdin).Encode(command); err != nil {
		return fmt.Errorf("could not request Pi entries: %w", err)
	}

	// utils
	var response struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Entries []struct {
				ID        string `json:"id"`
				Type      string `json:"type"`
				Timestamp string `json:"timestamp"`
				Name      string `json:"name"`
				Message   struct {
					Role         string          `json:"role"`
					ErrorMessage string          `json:"errorMessage"`
					Content      json.RawMessage `json:"content"`
					Model        string          `json:"model"`
					ToolCallID   string          `json:"toolCallId"`
					ToolName     string          `json:"toolName"`
					IsError      bool            `json:"isError"`
				} `json:"message"`
			} `json:"entries"`
			LeafID *string `json:"leafId"`
		} `json:"data"`
	}
	for {
		if err := conn.Decoder.Decode(&response); err != nil {
			return fmt.Errorf("could not read Pi entries: %w", err)
		}
		if response.Type != "response" || response.ID != id {
			continue
		}
		if !response.Success {
			return fmt.Errorf("Pi refused entries: %s", response.Error)
		}
		break
	}

	// acutally poll entries
	events := make([]harness.Event, 0)
	for _, entry := range response.Data.Entries {
		conn.Cursor = entry.ID
		if entry.Type == "session_info" && entry.Name != "" {
			s.Mu.Lock()
			s.Name = entry.Name
			s.Mu.Unlock()
		}
		if entry.Type != "message" {
			continue
		}
		timestamp := int64(0)
		if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
			timestamp = t.UnixMilli()
		}
		var content string
		var images []string
		if json.Unmarshal(entry.Message.Content, &content) != nil {
			var parts []struct {
				Type      string          `json:"type"`
				Text      string          `json:"text"`
				Thinking  string          `json:"thinking"`
				ID        string          `json:"id"`
				Name      string          `json:"name"`
				Arguments json.RawMessage `json:"arguments"`
				MimeType  string          `json:"mimeType"`
				Data      string          `json:"data"`
			}
			if json.Unmarshal(entry.Message.Content, &parts) == nil {
				for _, part := range parts {
					switch part.Type {
					case "text":
						content += part.Text
					case "image":
						if uri := harness.DataURI(part.MimeType, part.Data); uri != "" {
							images = append(images, uri)
						}
					case "thinking":
						if text := strings.TrimSpace(part.Thinking); text != "" && entry.Message.Role == "assistant" {
							events = append(events, harness.Event{Type: harness.EventThinking, Content: text, TimestampMs: timestamp, Model: entry.Message.Model})
						}
					case "toolCall":
						var args string
						if len(part.Arguments) > 0 && string(part.Arguments) != "null" {
							args = string(part.Arguments)
						}
						events = append(events, harness.Event{Type: harness.EventToolStart, ToolName: part.Name, ToolCallID: part.ID, Content: args, TimestampMs: timestamp})
					}
				}
			}
		}
		switch entry.Message.Role {
		case "toolResult":
			var resultParts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			var result string
			if json.Unmarshal(entry.Message.Content, &resultParts) == nil {
				for _, part := range resultParts {
					if part.Type == "text" {
						result += part.Text
					}
				}
			}
			events = append(events, harness.Event{Type: harness.EventToolEnd, ToolName: entry.Message.ToolName, ToolCallID: entry.Message.ToolCallID, Content: strings.TrimSpace(result), Images: images, ToolFailed: entry.Message.IsError, TimestampMs: timestamp})
		case "user":
			if content = strings.TrimSpace(content); content != "" || len(images) > 0 {
				events = append(events, harness.Event{Type: harness.EventUser, Content: content, Images: images, TimestampMs: timestamp})
			}
		case "assistant":
			if entry.Message.ErrorMessage != "" {
				events = append(events, harness.Event{Type: harness.EventError, Content: "Model response failed: " + entry.Message.ErrorMessage, TimestampMs: timestamp})
			} else if content = strings.TrimSpace(content); content != "" || len(images) > 0 {
				events = append(events, harness.Event{Type: harness.EventAssistant, Content: content, Images: images, TimestampMs: timestamp, Model: entry.Message.Model})
			}
		}
	}

	// poll stats
	statsID := id + "-stats"
	if err := json.NewEncoder(conn.Stdin).Encode(map[string]any{"id": statsID, "type": "get_session_stats"}); err != nil {
		return fmt.Errorf("could not request Pi stats: %w", err)
	}
	var statsResponse struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			Tokens struct {
				Input      int64 `json:"input"`
				Output     int64 `json:"output"`
				CacheRead  int64 `json:"cacheRead"`
				CacheWrite int64 `json:"cacheWrite"`
			} `json:"tokens"`
			Cost         float64 `json:"cost"`
			ContextUsage struct {
				Tokens        *int64 `json:"tokens"`
				ContextWindow int64  `json:"contextWindow"`
			} `json:"contextUsage"`
		} `json:"data"`
	}
	for {
		if err := conn.Decoder.Decode(&statsResponse); err != nil {
			return fmt.Errorf("could not read Pi stats: %w", err)
		}
		if statsResponse.Type != "response" || statsResponse.ID != statsID {
			continue
		}
		if !statsResponse.Success {
			return fmt.Errorf("Pi refused stats: %s", statsResponse.Error)
		}
		break
	}

	// getting busy state
	stateID := id + "-state"
	if err := json.NewEncoder(conn.Stdin).Encode(map[string]any{"id": stateID, "type": "get_state"}); err != nil {
		return fmt.Errorf("could not request Pi state: %w", err)
	}
	var stateResponse struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Success bool   `json:"success"`
		Error   string `json:"error"`
		Data    struct {
			IsStreaming  bool `json:"isStreaming"`
			IsCompacting bool `json:"isCompacting"`
		} `json:"data"`
	}
	for {
		if err := conn.Decoder.Decode(&stateResponse); err != nil {
			return fmt.Errorf("could not read Pi state: %w", err)
		}
		if stateResponse.Type != "response" || stateResponse.ID != stateID {
			continue
		}
		if !stateResponse.Success {
			return fmt.Errorf("Pi refused state: %s", stateResponse.Error)
		}
		break
	}
	busyCli := stateResponse.Data.IsStreaming || stateResponse.Data.IsCompacting

	// update events and stats from Pi's session stats response
	s.Mu.Lock()
	s.BusyCli = busyCli
	if full {
		s.Events = events
	} else {
		s.Events = append(s.Events, events...)
	}
	if s.FirstUserMsg == "" {
		for _, event := range events {
			if event.Type == harness.EventUser && event.Content != "" {
				s.FirstUserMsg = event.Content
				break
			}
		}
	}
	s.FullInputToken = statsResponse.Data.Tokens.Input
	s.FullOutputToken = statsResponse.Data.Tokens.Output
	s.CacheReadToken = statsResponse.Data.Tokens.CacheRead
	s.CacheWriteToken = statsResponse.Data.Tokens.CacheWrite
	s.FullCost = statsResponse.Data.Cost
	s.ContextTokens = 0
	if statsResponse.Data.ContextUsage.Tokens != nil {
		s.ContextTokens = *statsResponse.Data.ContextUsage.Tokens
	}
	s.ContextWindow = statsResponse.Data.ContextUsage.ContextWindow
	for _, entry := range response.Data.Entries {
		if entry.Type == "message" {
			if timestamp, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil && timestamp.UnixMilli() > s.LastTimestampMs {
				s.LastTimestampMs = timestamp.UnixMilli()
			}
		}
	}
	s.Mu.Unlock()
	return nil
}

// === utils fns

func (*Adapter) conn(s *harness.Session) (*connStruct, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if conn := cacheConns[s.RealPath][s.ID]; conn != nil {
		return conn, nil
	}
	return nil, errors.New("Pi session is not running")
}
