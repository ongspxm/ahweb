package claude

import (
	"bufio"
	"bytes"
	"encoding/hex"
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

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error { _, err := exec.LookPath("claude"); return err }

type conn struct {
	cmd    *exec.Cmd
	stderr bytes.Buffer

	mu                  sync.Mutex
	events              []harness.Event
	usageInputTkns      int64
	usageOutputTkns     int64
	usageCacheReadTkns  int64
	usageCacheWriteTkns int64
	usageCostUSD        float64
	contextTokens       int64
	contextWindow       int64

	// isExist=true when the transcript exist in claude
	// isStopped=true when "Stop()" is called, tell cmd error VS cmd purposely stopped
	isExist   bool
	isStopped bool
}

var (
	// cache[realPath][sessid]
	cacheMu sync.Mutex
	cache   = make(map[string]map[string]*conn)
)

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	if _, err := exec.LookPath("claude"); err != nil {
		return nil, fmt.Errorf("Claude Code CLI is not installed: %w", err)
	}
	cmd := exec.Command("claude", "-p", "--input-format", "stream-json", "--output-format", "stream-json", "--verbose")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Claude input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Claude output: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start Claude Code: %w", err)
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
	if _, err := io.WriteString(stdin, `{"type":"control_request","request_id":"models","request":{"subtype":"initialize"}}`+"\n"); err != nil {
		return nil, fmt.Errorf("could not ask Claude for models: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		var msg struct {
			Type     string `json:"type"`
			Response struct {
				Response struct {
					Models []struct {
						Value                 string   `json:"value"`
						ResolvedModel         string   `json:"resolvedModel"`
						SupportedEffortLevels []string `json:"supportedEffortLevels"`
					} `json:"models"`
				} `json:"response"`
			} `json:"response"`
		}
		if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.Type != "control_response" {
			continue
		}
		models := make(map[string]harness.Model)
		for _, m := range msg.Response.Response.Models {
			name := m.ResolvedModel
			if name == "" {
				name = m.Value
			}
			id := "claude/" + name
			models[id] = harness.Model{Harness: "claude", ID: id, Reasoning: m.SupportedEffortLevels}
		}
		if len(models) == 0 {
			return nil, errors.New("Claude Code returned no models")
		}
		return models, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read Claude models: %w", err)
	}
	return nil, errors.New("could not read Claude models")
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	dir := filepath.Dir(transcriptPath(realFolder, ""))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []harness.SessionLite{}, nil
		}
		return nil, fmt.Errorf("could not read Claude sessions: %w", err)
	}
	var sessions []harness.SessionLite
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		nativeID := strings.TrimSuffix(entry.Name(), ".jsonl")
		localID := nativeID
		cacheMu.Lock()
		for id := range cache[realFolder] {
			if sessionUUID(id) == nativeID {
				localID = id
				break
			}
		}
		cacheMu.Unlock()
		lite := harness.SessionLite{ID: localID, RealPath: realFolder, Harness: "claude"}
		f, err := os.Open(transcriptPath(realFolder, nativeID))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			var row struct {
				Timestamp string `json:"timestamp"`
				Message   struct {
					Role    string          `json:"role"`
					Content json.RawMessage `json:"content"`
				} `json:"message"`
			}
			if json.Unmarshal(scanner.Bytes(), &row) != nil || (row.Message.Role != "user" && row.Message.Role != "assistant") {
				continue
			}
			if parsed, err := time.Parse(time.RFC3339Nano, row.Timestamp); err == nil && parsed.UnixMilli() > lite.LastTimestampMs {
				lite.LastTimestampMs = parsed.UnixMilli()
			}
			if row.Message.Role == "user" && lite.FirstUserMsg == "" {
				if json.Unmarshal(row.Message.Content, &lite.FirstUserMsg) != nil {
					var parts []struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}
					if json.Unmarshal(row.Message.Content, &parts) == nil {
						for _, part := range parts {
							if part.Type == "text" && part.Text != "" {
								lite.FirstUserMsg = part.Text
								break
							}
						}
					}
				}
			}
		}
		f.Close()
		if lite.FirstUserMsg != "" || lite.LastTimestampMs != 0 {
			sessions = append(sessions, lite)
		}
	}
	return sessions, nil
}

// === session admin fns

func (a *Adapter) Connect(s *harness.Session, _ bool) error {
	if _, err := exec.LookPath("claude"); err != nil {
		return fmt.Errorf("Claude Code CLI is not installed: %w", err)
	}
	cacheMu.Lock()
	if cache[s.RealPath] == nil {
		cache[s.RealPath] = make(map[string]*conn)
	}
	c := cache[s.RealPath][s.ID]
	if c == nil {
		sessionID := sessionUUID(s.ID)
		c = &conn{events: append([]harness.Event(nil), s.Events...)}
		if _, err := os.Stat(transcriptPath(s.RealPath, sessionID)); err == nil {
			c.isExist = true
		}
		cache[s.RealPath][s.ID] = c
	}
	cacheMu.Unlock()
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
	cmd := c.cmd
	c.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
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
	prompt, images, model, effort := s.SendingMsg, append([]string(nil), s.SendingImages...), s.PromptModel, s.PromptReasoning
	s.Mu.Unlock()
	parts := make([]any, 0, len(images)+1)
	if prompt != "" {
		parts = append(parts, map[string]string{"type": "text", "text": prompt})
	}
	for _, uri := range images {
		image, err := harness.ParseImage(uri)
		if err != nil {
			return err
		}
		parts = append(parts, map[string]any{"type": "image", "source": map[string]string{"type": "base64", "media_type": image.MimeType, "data": image.Data}})
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cmd != nil {
		return errors.New("Claude session is busy")
	}
	args := []string{"-p", "--dangerously-skip-permissions", "--output-format", "stream-json", "--verbose"}
	if len(images) > 0 {
		args = append(args, "--input-format", "stream-json")
	}
	if c.isExist {
		args = append(args, "--resume", sessionUUID(s.ID))
	} else {
		args = append(args, "--session-id", sessionUUID(s.ID))
	}

	if model != nil && model.ID != "" {
		modelID := strings.TrimPrefix(model.ID, "claude/")
		args = append(args, "--model", modelID)
	}
	if effort != "" {
		args = append(args, "--effort", effort)
	}
	if len(images) == 0 {
		args = append(args, prompt)
	}
	cmd := exec.Command("claude", args...)
	cmd.Dir = s.RealPath
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("could not open Claude output: %w", err)
	}
	var stdin io.WriteCloser
	if len(images) > 0 {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return fmt.Errorf("could not open Claude input: %w", err)
		}
	}

	c.stderr.Reset()
	cmd.Stderr = &c.stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("could not start Claude Code: %w", err)
	}
	if stdin != nil {
		err := json.NewEncoder(stdin).Encode(map[string]any{"type": "user", "message": map[string]any{"role": "user", "content": parts}})
		_ = stdin.Close()
		if err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fmt.Errorf("could not send Claude image prompt: %w", err)
		}
	}

	c.cmd, c.isStopped = cmd, false
	c.events = append(c.events, harness.Event{Type: harness.EventUser, Content: prompt, Images: images, TimestampMs: time.Now().UnixMilli()})
	s.Mu.Lock()
	s.BusyCli = true
	s.Mu.Unlock()

	// the background thread that lists for live events
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			var msg map[string]any
			if json.Unmarshal(scanner.Bytes(), &msg) != nil {
				continue
			}
			kind, _ := msg["type"].(string)
			if kind == "system" {
				if subtype, _ := msg["subtype"].(string); subtype == "init" {
					c.mu.Lock()
					c.isExist = true
					c.mu.Unlock()
				}
			}
			_add := func(e harness.Event) {
				e.TimestampMs = time.Now().UnixMilli()
				c.mu.Lock()
				if e.Type == harness.EventToolStart && e.ToolCallID != "" {
					for _, existing := range c.events {
						if existing.Type == e.Type && existing.ToolCallID == e.ToolCallID {
							c.mu.Unlock()
							return
						}
					}
				}
				c.events = append(c.events, e)
				c.mu.Unlock()
			}
			content, _ := msg["content"].([]any)
			if message, ok := msg["message"].(map[string]any); ok {
				content, _ = message["content"].([]any)
			}
			if kind == "stream_event" {
				inner, _ := msg["event"].(map[string]any)
				if inner["type"] == "content_block_start" {
					block, _ := inner["content_block"].(map[string]any)
					if block["type"] == "tool_use" {
						name, _ := block["name"].(string)
						id, _ := block["id"].(string)
						_add(harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: id})
					}
				}
				continue
			}
			if kind == "assistant" || kind == "user" {
				if kind == "assistant" {
					message, _ := msg["message"].(map[string]any)
					model, _ := message["model"].(string)
					usage, _ := message["usage"].(map[string]any)
					var contextTokens int64
					for _, key := range []string{"input_tokens", "output_tokens", "cache_read_input_tokens", "cache_creation_input_tokens"} {
						if n, ok := usage[key].(float64); ok {
							contextTokens += int64(n)
						}
					}
					c.mu.Lock()
					c.contextTokens = contextTokens
					if window := modelContextWindow(model); window > 0 {
						c.contextWindow = window
					}
					c.mu.Unlock()
				}
				for _, part := range content {
					block, _ := part.(map[string]any)
					blockType, _ := block["type"].(string)
					text, _ := block["text"].(string)
					if kind == "assistant" && blockType == "text" && text != "" {
						_add(harness.Event{Type: harness.EventAssistant, Content: text})
					} else if kind == "assistant" && blockType == "thinking" && text != "" {
						_add(harness.Event{Type: harness.EventThinking, Content: text})
					} else if kind == "assistant" && blockType == "tool_use" {
						name, _ := block["name"].(string)
						id, _ := block["id"].(string)
						_add(harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: id})
					} else if image := imageDataURI(block); image != "" && kind == "assistant" {
						_add(harness.Event{Type: harness.EventAssistant, Images: []string{image}})
					}
					if blockType == "tool_result" {
						id, _ := block["tool_use_id"].(string)
						failed, _ := block["is_error"].(bool)
						text := ""
						if v, ok := block["content"].(string); ok {
							text = v
						} else if parts, ok := block["content"].([]any); ok {
							for _, part := range parts {
								if item, ok := part.(map[string]any); ok {
									if v, ok := item["text"].(string); ok {
										text += v
									}
								}
							}
						}
						_add(harness.Event{Type: harness.EventToolEnd, ToolCallID: id, Content: text, ToolFailed: failed})
					}
				}
			}
			if kind == "result" {
				if subtype, _ := msg["subtype"].(string); strings.Contains(subtype, "error") {
					text, _ := msg["result"].(string)
					_add(harness.Event{Type: harness.EventError, Content: text})
				}
				usage, _ := msg["usage"].(map[string]any)
				c.mu.Lock()
				for key, value := range usage {
					n, ok := value.(float64)
					if !ok {
						continue
					}
					switch key {
					case "input_tokens":
						c.usageInputTkns += int64(n)
					case "output_tokens":
						c.usageOutputTkns += int64(n)
					case "cache_read_input_tokens":
						c.usageCacheReadTkns += int64(n)
					case "cache_creation_input_tokens":
						c.usageCacheWriteTkns += int64(n)
					}
				}
				if cost, ok := msg["total_cost_usd"].(float64); ok {
					c.usageCostUSD = cost
				}
				c.mu.Unlock()
			}
		}
		scanErr := scanner.Err()
		waitErr := cmd.Wait()
		c.mu.Lock()
		message := strings.TrimSpace(c.stderr.String())
		interrupted := c.isStopped
		c.isStopped = false
		if scanErr != nil {
			c.events = append(c.events, harness.Event{Type: harness.EventError, Content: fmt.Sprintf("could not read Claude output: %v", scanErr), TimestampMs: time.Now().UnixMilli()})
		} else if waitErr != nil && !interrupted {
			if message == "" {
				message = waitErr.Error()
			}
			c.events = append(c.events, harness.Event{Type: harness.EventError, Content: "Claude Code failed: " + message, TimestampMs: time.Now().UnixMilli()})
		}
		c.cmd = nil
		c.mu.Unlock()
		s.Mu.Lock()
		s.BusyCli = false
		s.Mu.Unlock()
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

	// evts from full history
	if fullHistory {
		c.mu.Lock()
		sessionID := sessionUUID(s.ID)
		if _, err := os.Stat(transcriptPath(s.RealPath, sessionID)); err == nil {
			if f, err := os.Open(transcriptPath(s.RealPath, sessionID)); err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
				var events []harness.Event
				var input, output, cacheRead, cacheWrite int64
				var cost float64
				var hasCost bool
				var contextTokens, contextWindow int64
				seenMessageIDs := make(map[string]bool)
				for scanner.Scan() {
					var row struct {
						Type         string  `json:"type"`
						Timestamp    string  `json:"timestamp"`
						TotalCostUSD float64 `json:"total_cost_usd"`
						Message      struct {
							ID      string          `json:"id"`
							Model   string          `json:"model"`
							Role    string          `json:"role"`
							Content json.RawMessage `json:"content"`
							Usage   struct {
								InputTokens              int64 `json:"input_tokens"`
								OutputTokens             int64 `json:"output_tokens"`
								CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
								CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
							} `json:"usage"`
						} `json:"message"`
					}
					if json.Unmarshal(scanner.Bytes(), &row) != nil {
						continue
					}
					if row.TotalCostUSD > 0 {
						cost, hasCost = row.TotalCostUSD, true
					}
					role := row.Message.Role
					if role != "user" && role != "assistant" {
						continue
					}
					timestamp := int64(0)
					if parsed, err := time.Parse(time.RFC3339Nano, row.Timestamp); err == nil {
						timestamp = parsed.UnixMilli()
					}
					var text string
					if json.Unmarshal(row.Message.Content, &text) == nil {
						events = append(events, harness.Event{Type: eventType(role), Content: text, TimestampMs: timestamp})
					} else {
						var parts []map[string]any
						if json.Unmarshal(row.Message.Content, &parts) == nil {
							for _, part := range parts {
								kind, _ := part["type"].(string)
								content, _ := part["text"].(string)
								switch kind {
								case "text":
									if content != "" {
										events = append(events, harness.Event{Type: eventType(role), Content: content, TimestampMs: timestamp})
									}
								case "thinking":
									if content != "" {
										events = append(events, harness.Event{Type: harness.EventThinking, Content: content, TimestampMs: timestamp})
									}
								case "image":
									if image := imageDataURI(part); image != "" {
										events = append(events, harness.Event{Type: eventType(role), Images: []string{image}, TimestampMs: timestamp})
									}
								case "tool_use":
									name, _ := part["name"].(string)
									id, _ := part["id"].(string)
									events = append(events, harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: id, TimestampMs: timestamp})
								case "tool_result":
									id, _ := part["tool_use_id"].(string)
									failed, _ := part["is_error"].(bool)
									events = append(events, harness.Event{Type: harness.EventToolEnd, ToolCallID: id, ToolFailed: failed, TimestampMs: timestamp})
								}
							}
						}
					}
					if role == "assistant" && (row.Message.ID == "" || !seenMessageIDs[row.Message.ID]) {
						if row.Message.ID != "" {
							seenMessageIDs[row.Message.ID] = true
						}
						input += row.Message.Usage.InputTokens
						output += row.Message.Usage.OutputTokens
						cacheRead += row.Message.Usage.CacheReadInputTokens
						cacheWrite += row.Message.Usage.CacheCreationInputTokens
					}
					if role == "assistant" {
						usage := row.Message.Usage
						contextTokens = usage.InputTokens + usage.OutputTokens + usage.CacheReadInputTokens + usage.CacheCreationInputTokens
						contextWindow = modelContextWindow(row.Message.Model)
					}
				}
				c.events = events
				c.usageInputTkns = input
				c.usageOutputTkns = output
				c.usageCacheReadTkns = cacheRead
				c.usageCacheWriteTkns = cacheWrite
				if hasCost {
					c.usageCostUSD = cost
				}
				c.contextTokens, c.contextWindow = contextTokens, contextWindow
			}
		}
		c.mu.Unlock()
	}

	// evts from live feed
	c.mu.Lock()
	events := append([]harness.Event(nil), c.events...)
	busy := c.cmd != nil
	input, output, cacheRead, cacheWrite, cost, contextTokens, contextWindow := c.usageInputTkns, c.usageOutputTkns, c.usageCacheReadTkns, c.usageCacheWriteTkns, c.usageCostUSD, c.contextTokens, c.contextWindow
	c.mu.Unlock()

	s.Mu.Lock()
	s.Events = events
	for _, event := range events {
		if event.TimestampMs > s.LastTimestampMs {
			s.LastTimestampMs = event.TimestampMs
		}
	}
	s.BusyCli = busy
	if len(s.Events) > 0 && s.FirstUserMsg == "" {
		for _, e := range s.Events {
			if e.Type == harness.EventUser {
				s.FirstUserMsg = e.Content
				break
			}
		}
	}
	s.FullInputToken = input
	s.FullOutputToken = output
	s.CacheReadToken = cacheRead
	s.CacheWriteToken = cacheWrite
	s.FullCost = cost
	s.ContextTokens, s.ContextWindow = contextTokens, contextWindow
	s.DoFullHistory = false
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
		return fmt.Errorf("could not interrupt Claude Code: %w", err)
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
	return nil, errors.New("Claude session is not connected")
}

func eventType(role string) harness.EventType {
	if role == "user" {
		return harness.EventUser
	}
	return harness.EventAssistant
}

func modelContextWindow(model string) int64 {
	model = strings.TrimPrefix(model, "claude/")
	if strings.Contains(model, "sonnet-5") || strings.Contains(model, "opus-5") || strings.Contains(model, "opus-4-8") || strings.Contains(model, "opus-4-7") || strings.Contains(model, "opus-4-6") || strings.Contains(model, "sonnet-4-6") {
		return 1_000_000
	}
	if strings.HasPrefix(model, "claude-") {
		return 200_000
	}
	return 0
}

func imageDataURI(block map[string]any) string {
	source, _ := block["source"].(map[string]any)
	mimeType, _ := source["media_type"].(string)
	data, _ := source["data"].(string)
	return harness.DataURI(mimeType, data)
}

func sessionUUID(id string) string {
	if strings.Contains(id, "-") {
		return id
	}
	decoded, err := hex.DecodeString(id)
	if err != nil || len(decoded) != 16 {
		return id
	}
	decoded[6] = decoded[6]&0x0f | 0x40
	decoded[8] = decoded[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", decoded[:4], decoded[4:6], decoded[6:8], decoded[8:10], decoded[10:])
}

func transcriptPath(realFolder, sessionID string) string {
	home, _ := os.UserHomeDir()
	encoded := strings.TrimLeft(strings.ReplaceAll(realFolder, "\\", "/"), "/")
	encoded = strings.ReplaceAll(strings.ReplaceAll(encoded, "/", "-"), ".", "-")
	return filepath.Join(home, ".claude", "projects", "-"+encoded, sessionID+".jsonl")
}
