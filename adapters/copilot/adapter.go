package copilot

import (
	"bufio"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"

	_ "github.com/mattn/go-sqlite3"
)

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error {
	_, err := exec.LookPath("copilot")
	return err
}

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	_parseCopilotACPModels := func(raw []byte) (map[string]harness.Model, error) {
		var response struct {
			Result struct {
				Models struct {
					Available []struct {
						ModelID string         `json:"modelId"`
						Meta    map[string]any `json:"_meta"`
					} `json:"availableModels"`
				} `json:"models"`
				ConfigOptions []struct {
					ID      string `json:"id"`
					Options []struct {
						Value string         `json:"value"`
						Meta  map[string]any `json:"_meta"`
					} `json:"options"`
				} `json:"configOptions"`
			} `json:"result"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			return nil, fmt.Errorf("could not decode Copilot ACP model catalog: %w", err)
		}

		reasoning := []string{}
		for _, option := range response.Result.ConfigOptions {
			if option.ID != "reasoning_effort" {
				continue
			}
			for _, level := range option.Options {
				if level.Value != "" {
					reasoning = append(reasoning, level.Value)
				}
			}
		}

		models := map[string]harness.Model{}
		for _, option := range response.Result.ConfigOptions {
			if option.ID != "model" {
				continue
			}
			for _, modelOption := range option.Options {
				if enabled, ok := modelOption.Meta["copilotEnablement"].(string); ok && enabled != "enabled" {
					continue
				}
				if modelOption.Value == "" {
					continue
				}
				id := "copilot/" + modelOption.Value
				models[id] = harness.Model{Harness: "copilot", ID: id, Reasoning: append([]string(nil), reasoning...)}
			}
		}
		for _, model := range response.Result.Models.Available {
			if enabled, ok := model.Meta["copilotEnablement"].(string); ok && enabled != "enabled" {
				continue
			}
			if model.ModelID == "" {
				continue
			}
			id := "copilot/" + model.ModelID
			models[id] = harness.Model{Harness: "copilot", ID: id, Reasoning: append([]string(nil), reasoning...)}
		}
		if len(models) == 0 {
			return nil, errors.New("Copilot ACP returned no valid models")
		}
		return models, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get working directory for Copilot model catalog: %w", err)
	}
	cmd := exec.Command("copilot", "--acp")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Copilot ACP input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Copilot ACP output: %w", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start Copilot ACP: %w", err)
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()
	timer := time.AfterFunc(30*time.Second, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	defer timer.Stop()

	if _, err := fmt.Fprintln(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{}}}`); err != nil {
		return nil, fmt.Errorf("could not initialize Copilot ACP: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
	for scanner.Scan() {
		var msg struct {
			ID int `json:"id"`
		}
		if json.Unmarshal(scanner.Bytes(), &msg) == nil && msg.ID == 1 {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read Copilot ACP initialization: %w", err)
	}

	params, err := json.Marshal(map[string]any{"cwd": cwd, "mcpServers": []any{}})
	if err != nil {
		return nil, fmt.Errorf("could not prepare Copilot ACP session: %w", err)
	}
	request, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "session/new",
		"params":  json.RawMessage(params),
	})
	if err != nil {
		return nil, fmt.Errorf("could not prepare Copilot ACP model request: %w", err)
	}
	if _, err := fmt.Fprintln(stdin, string(request)); err != nil {
		return nil, fmt.Errorf("could not ask Copilot ACP for models: %w", err)
	}
	for scanner.Scan() {
		var msg struct {
			ID int `json:"id"`
		}
		if json.Unmarshal(scanner.Bytes(), &msg) != nil || msg.ID != 2 {
			continue
		}
		models, err := _parseCopilotACPModels(scanner.Bytes())
		if err != nil {
			return nil, err
		}
		return models, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read Copilot ACP model catalog: %w", err)
	}
	return nil, errors.New("Copilot ACP returned no model catalog")
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	db, err := copilotDB()
	if err != nil {
		if os.IsNotExist(err) {
			return []harness.SessionLite{}, nil
		}
		return nil, fmt.Errorf("could not open Copilot session store: %w", err)
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT id, cwd, COALESCE(summary, ''), COALESCE(created_at, ''), COALESCE(updated_at, '')
		FROM sessions WHERE cwd = ? ORDER BY updated_at DESC`, realFolder)
	if err != nil {
		return nil, fmt.Errorf("could not list Copilot sessions: %w", err)
	}
	defer rows.Close()

	var sessions []harness.SessionLite
	seen := make(map[string]bool)
	for rows.Next() {
		var id, cwd, summary, created, updated string
		if err := rows.Scan(&id, &cwd, &summary, &created, &updated); err != nil {
			continue
		}
		localID := id
		cacheMu.Lock()
		if sessID := harnessIDToSessID[realFolder][id]; sessID != "" {
			localID = sessID
		}
		cacheMu.Unlock()
		if seen[localID] {
			continue
		}

		lite := harness.SessionLite{ID: localID, RealPath: cwd, Harness: "copilot", FirstUserMsg: summary}
		for _, value := range []string{updated, created} {
			if stamp, err := time.Parse(time.RFC3339Nano, value); err == nil {
				lite.LastTimestampMs = stamp.UnixMilli()
				break
			}
		}
		seen[localID] = true
		sessions = append(sessions, lite)
	}
	return sessions, rows.Err()
}

// === session admin fns

type conn struct {
	mu                     sync.Mutex
	cmd                    *exec.Cmd
	done                   chan struct{}
	harnessID              string
	busy, closing, started bool

	events, errors []harness.Event
	used, size     int64
}

var (
	cacheMu           sync.Mutex
	cache             = make(map[string]map[string]*conn)
	harnessIDToSessID = make(map[string]map[string]string)
	sessIDToHarnessID = make(map[string]map[string]string)
)

func (*Adapter) Connect(s *harness.Session, _ bool) error {
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
		harnessID := s.ID
		if !strings.Contains(harnessID, "-") {
			if decoded, err := hex.DecodeString(harnessID); err == nil && len(decoded) == 16 {
				decoded[6] = decoded[6]&0x0f | 0x40
				decoded[8] = decoded[8]&0x3f | 0x80
				harnessID = fmt.Sprintf("%x-%x-%x-%x-%x", decoded[:4], decoded[4:6], decoded[6:8], decoded[8:10], decoded[10:])
			}
		}
		harnessIDToSessID[s.RealPath][s.ID] = s.ID
		harnessIDToSessID[s.RealPath][harnessID] = s.ID
		sessIDToHarnessID[s.RealPath][s.ID] = harnessID
		cache[s.RealPath][s.ID] = &conn{harnessID: harnessID}
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
	c.closing = true
	cmd, done := c.cmd, c.done
	c.mu.Unlock()
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
	if done != nil {
		<-done
	}
	cacheMu.Lock()
	delete(cache[s.RealPath], s.ID)
	cacheMu.Unlock()
	return nil
}

func (*Adapter) CanDelete() bool { return true }

func (a *Adapter) Delete(s *harness.Session) error {
	if s.GetIsBusy() {
		return errors.New("cannot delete a busy Copilot session")
	}

	cacheMu.Lock()
	c := cache[s.RealPath][s.ID]
	harnessID := sessIDToHarnessID[s.RealPath][s.ID]
	cacheMu.Unlock()
	if c != nil {
		c.mu.Lock()
		if c.busy || c.closing {
			c.mu.Unlock()
			return errors.New("cannot delete a busy Copilot session")
		}
		c.mu.Unlock()
	}

	if err := a.CleanUp(s); err != nil {
		return err
	}
	stateDir := filepath.Join(copilotHome(), "session-state", harnessID)
	if err := os.RemoveAll(stateDir); err != nil {
		return fmt.Errorf("could not remove Copilot session state: %w", err)
	}

	db, err := copilotDB()
	if err != nil {
		if os.IsNotExist(err) {
			cacheMu.Lock()
			delete(harnessIDToSessID[s.RealPath], harnessID)
			delete(harnessIDToSessID[s.RealPath], s.ID)
			delete(sessIDToHarnessID[s.RealPath], s.ID)
			cacheMu.Unlock()
			return nil
		}
		return fmt.Errorf("could not open Copilot session store: %w", err)
	}
	defer db.Close()
	if _, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, harnessID); err != nil {
		return fmt.Errorf("could not remove Copilot session index: %w", err)
	}
	cacheMu.Lock()
	delete(harnessIDToSessID[s.RealPath], harnessID)
	delete(harnessIDToSessID[s.RealPath], s.ID)
	delete(sessIDToHarnessID[s.RealPath], s.ID)
	cacheMu.Unlock()
	return nil
}

// === session fns

func (a *Adapter) SendPrompt(s *harness.Session) error {
	c, err := a.conn(s)
	if err != nil {
		return err
	}

	// setup the cli cmd
	s.Mu.Lock()
	prompt, images, model, effort := s.SendingMsg, append([]string(nil), s.SendingImages...), s.PromptModel, s.PromptReasoning
	s.Mu.Unlock()
	args := []string{"-p", prompt, "--output-format", "json", "--stream", "on", "--allow-all"}
	c.mu.Lock()
	if c.busy || c.closing {
		c.mu.Unlock()
		return errors.New("Copilot session is busy")
	}
	id := c.harnessID
	if strings.Contains(s.ID, "-") || c.started {
		args = append(args, "--resume="+id)
	} else {
		args = append(args, "--session-id="+id)
	}
	if model != nil && model.ID != "" {
		args = append(args, "--model", strings.TrimPrefix(model.ID, "copilot/"))
	}
	if effort != "" {
		args = append(args, "--reasoning-effort", effort)
	}

	// form the actual imgs
	var attachments []string
	attachmentDir := filepath.Join(copilotHome(), "session-state", id, "attachments")
	keepAttachments := false
	defer func() {
		if !keepAttachments {
			for _, path := range attachments {
				_ = os.Remove(path)
			}
		}
	}()
	if len(images) > 0 {
		if e := os.MkdirAll(attachmentDir, 0700); e != nil {
			c.mu.Unlock()
			return fmt.Errorf("could not create Copilot attachment directory: %w", e)
		}
	}
	for _, uri := range images {
		image, e := harness.ParseImage(uri)
		if e != nil {
			c.mu.Unlock()
			return e
		}
		data, e := base64.StdEncoding.DecodeString(image.Data)
		if e != nil {
			c.mu.Unlock()
			return e
		}
		extensions, e := mime.ExtensionsByType(image.MimeType)
		if e != nil || len(extensions) == 0 {
			c.mu.Unlock()
			return fmt.Errorf("unsupported image MIME type: %s", image.MimeType)
		}
		extension := extensions[0]
		if image.MimeType == "image/jpeg" {
			extension = ".jpg"
		}
		file, e := os.CreateTemp(attachmentDir, "attachment-*"+extension)
		if e != nil {
			c.mu.Unlock()
			return e
		}
		if _, e = file.Write(data); e == nil {
			e = file.Close()
		} else {
			_ = file.Close()
		}
		if e != nil {
			_ = os.Remove(file.Name())
			c.mu.Unlock()
			return e
		}
		attachments = append(attachments, file.Name())
		args = append(args, "--attachment", file.Name())
	}

	// call the cmd and start parsing stdout
	cmd := exec.Command("copilot", args...)
	cmd.Dir, cmd.Stderr = s.RealPath, os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	if err = cmd.Start(); err != nil {
		c.mu.Unlock()
		return err
	}
	keepAttachments = true
	done := make(chan struct{})
	c.cmd, c.done, c.busy, c.started = cmd, done, true, true
	userEvent := harness.Event{Type: harness.EventUser, Content: prompt, Images: append([]string(nil), images...), TimestampMs: time.Now().UnixMilli()}
	c.mu.Unlock()

	// here we choose to add to the s.Events directly
	// claim: the cmd will add it to the events.jsonl immediately
	s.Mu.Lock()
	s.Events = append(s.Events, userEvent)
	s.BusyCli = true
	s.Mu.Unlock()

	// processing evts from stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			// here we skipUser because we manually added to s.Events directly
			msg, event, ok := parseJsonToEvent(scanner.Bytes(), true)
			if !ok {
				continue
			}

			c.mu.Lock()
			if event.Type != "" {
				c.events = append(c.events, event)
			}
			switch msg.Type {
			case "session.start":
				if id, _ := msg.Data["sessionId"].(string); id != "" {
					c.harnessID = id
					cacheMu.Lock()
					harnessIDToSessID[s.RealPath][id] = s.ID
					sessIDToHarnessID[s.RealPath][s.ID] = id
					cacheMu.Unlock()
				}
			case "session.usage_info":
				if used, ok := msg.Data["used"].(float64); ok {
					c.used = int64(used)
				}
				if size, ok := msg.Data["size"].(float64); ok {
					c.size = int64(size)
				}
			}
			c.mu.Unlock()
		}
		scanErr := scanner.Err()
		waitErr := cmd.Wait()
		c.mu.Lock()
		if !c.closing {
			if scanErr != nil {
				c.errors = append(c.errors, harness.Event{Type: harness.EventError, Content: scanErr.Error(), TimestampMs: time.Now().UnixMilli()})
			}
			if waitErr != nil {
				c.errors = append(c.errors, harness.Event{Type: harness.EventError, Content: waitErr.Error(), TimestampMs: time.Now().UnixMilli()})
			}
		}
		c.cmd, c.busy = nil, false
		c.mu.Unlock()
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

	// SQLite provides usage totals; events.jsonl provides conversation history.
	c.mu.Lock()
	busy := c.busy
	harnessID := c.harnessID
	c.mu.Unlock()

	_readSessionEvents := func(id string) []harness.Event {
		file, err := os.Open(filepath.Join(copilotHome(), "session-state", id, "events.jsonl"))
		if err != nil {
			return nil
		}
		defer file.Close()

		var events []harness.Event
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			_, event, ok := parseJsonToEvent(scanner.Bytes(), false)
			if !ok {
				continue
			}
			if event.Type != "" {
				events = append(events, event)
			}
		}
		if scanner.Err() != nil {
			return nil
		}
		return events
	}

	// first we setup full events
	c.mu.Lock()
	var events []harness.Event
	if fullHistory {
		events = _readSessionEvents(harnessID)
	} else {
		s.Mu.Lock()
		events = append([]harness.Event(nil), s.Events...)
		s.Mu.Unlock()
	}
	events = append(append(events, c.events...), c.errors...)
	c.errors = nil
	c.events = nil
	used, size := c.used, c.size
	c.mu.Unlock()

	// then read the db usage details
	var dbInput, dbOutput, dbCacheRead, dbCacheWrite, totalNanoAiu int64
	dbCost := 0.0
	if db, err := copilotDB(); err == nil {
		defer db.Close()
		_ = db.QueryRow(`SELECT COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cache_read_tokens), 0), COALESCE(SUM(cache_write_tokens), 0) FROM assistant_usage_events WHERE session_id = ?`, harnessID).Scan(&dbInput, &dbOutput, &dbCacheRead, &dbCacheWrite)
		_ = db.QueryRow(`SELECT COALESCE(SUM(total_nano_aiu), 0) FROM assistant_usage_events WHERE session_id = ?`, harnessID).Scan(&totalNanoAiu)
		dbCost = float64(totalNanoAiu) / 1_000_000_000 * 0.01
	}

	// then patch up the session stats
	s.Mu.Lock()
	s.Events, s.BusyCli, s.DoFullHistory = events, busy, false
	s.FullInputToken, s.FullOutputToken = dbInput, dbOutput
	s.CacheReadToken, s.CacheWriteToken = dbCacheRead, dbCacheWrite
	s.FullCost = dbCost
	s.ContextTokens, s.ContextWindow = used, size
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
	defer c.mu.Unlock()
	if c.cmd != nil && c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// === util fns

func (*Adapter) conn(s *harness.Session) (*conn, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if c := cache[s.RealPath][s.ID]; c != nil {
		return c, nil
	}
	return nil, errors.New("Copilot session is not connected")
}

func copilotHome() string {
	if value := strings.TrimSpace(os.Getenv("COPILOT_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".copilot")
	}
	return ".copilot"
}

func copilotDB() (*sql.DB, error) {
	path := filepath.Join(copilotHome(), "session-store.db")
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite3", "file:"+path+"?mode=rw")
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		return nil, err
	}
	return db, nil
}

type copilotMsg struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

func parseJsonToEvent(line []byte, skipUser bool) (copilotMsg, harness.Event, bool) {
	var msg copilotMsg
	if json.Unmarshal(line, &msg) != nil || msg.Data == nil {
		return msg, harness.Event{}, false
	}
	if skipUser && msg.Type == "user.message" {
		return msg, harness.Event{}, false
	}
	stamp := int64(0)
	if parsed, err := time.Parse(time.RFC3339Nano, msg.Timestamp); err == nil {
		stamp = parsed.UnixMilli()
	}
	if stamp == 0 {
		stamp = time.Now().UnixMilli()
	}
	var event harness.Event
	switch msg.Type {
	case "user.message", "assistant.message":
		// here we choose to ignore message_delta, we checked
		// assistant.message always follows message_delta
		content, _ := msg.Data["content"].(string)
		images := []string{}
		if msg.Type == "user.message" {
			if attachments, ok := msg.Data["attachments"].([]any); ok {
				for _, raw := range attachments {
					attachment, ok := raw.(map[string]any)
					if !ok {
						continue
					}
					mimeType, _ := attachment["mimeType"].(string)
					if !strings.HasPrefix(mimeType, "image/") {
						continue
					}
					path, _ := attachment["path"].(string)
					data, err := os.ReadFile(path)
					if err != nil {
						continue
					}
					if uri := harness.DataURI(mimeType, base64.StdEncoding.EncodeToString(data)); uri != "" {
						images = append(images, uri)
					}
				}
			}
		}
		if strings.TrimSpace(content) != "" || len(images) > 0 {
			typ := harness.EventAssistant
			if msg.Type == "user.message" {
				typ = harness.EventUser
			}
			event = harness.Event{Type: typ, Content: content, Images: images, TimestampMs: stamp}
		}
	case "tool.execution_start":
		callID, _ := msg.Data["toolCallId"].(string)
		name, _ := msg.Data["toolName"].(string)
		if name == "" {
			name, _ = msg.Data["toolTitle"].(string)
		}
		detail, _ := msg.Data["command"].(string)
		switch args := msg.Data["arguments"].(type) {
		case map[string]any:
			command, _ := args["command"].(string)
			description, _ := args["description"].(string)
			if command != "" {
				detail = command
			} else if description != "" {
				detail = description
			} else if encoded, err := json.Marshal(args); err == nil {
				detail = string(encoded)
			}
		case string:
			detail = args
		}
		event = harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: callID, Content: detail, TimestampMs: stamp}
	case "tool.execution_complete":
		callID, _ := msg.Data["toolCallId"].(string)
		result, _ := msg.Data["result"].(map[string]any)
		content, _ := result["content"].(string)
		if content == "" {
			content, _ = msg.Data["output"].(string)
		}
		failed := false
		if success, ok := msg.Data["success"].(bool); ok {
			failed = !success
		} else {
			exitCode, _ := msg.Data["exitCode"].(float64)
			if shell, ok := msg.Data["shellExecution"].(map[string]any); ok {
				exitCode, _ = shell["exitCode"].(float64)
			}
			failed = exitCode != 0
		}
		event = harness.Event{Type: harness.EventToolEnd, ToolCallID: callID, ToolFailed: failed, Content: content, TimestampMs: stamp}
	}
	return msg, event, true
}
