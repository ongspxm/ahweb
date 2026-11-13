package prime

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
	"syscall"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type SessionConn struct {
	Mu              sync.Mutex
	MsgCursor       int
	EventSequence   uint64
	EventGeneration string
	ActiveSessionID string
}

const (
	daemonProtocolName    = "prime-agent.daemon"
	daemonProtocolVersion = 7
	daemonClientID        = "ahweb"
)

var (
	errDaemonUnavailable  = errors.New("prime daemon is not available")
	errDaemonIncompatible = errors.New("prime daemon protocol is not compatible")

	sessionConnsMu sync.Mutex
	sessionConns   = make(map[string]map[string]*SessionConn)

	daemonStartMu sync.Mutex
)

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error {
	// if daemon start, then skip, else start daemon
	if _, err := exec.LookPath("prime-agent"); err != nil {
		return err
	}
	_socketPath := func() string {
		if socket := strings.TrimSpace(os.Getenv("PRIME_AGENT_DAEMON_SOCKET")); socket != "" {
			return socket
		}
		return filepath.Join(os.TempDir(), fmt.Sprintf("prime-agent-rust-%d", os.Getuid()), "daemon.sock")
	}
	_dial := func(c *connBase) error {
		var lastErr error
		for attempt := range 3 {
			if attempt > 0 {
				time.Sleep(time.Duration(attempt) * 100 * time.Millisecond)
			}
			c.mu.Lock()
			c.retryAt = time.Time{}
			_, err := c.ensureLocked()
			c.mu.Unlock()
			if err == nil {
				return nil
			}
			if errors.Is(err, errDaemonIncompatible) {
				return err
			}
			lastErr = err
		}
		return fmt.Errorf("%w: %v", errDaemonUnavailable, lastErr)
	}

	daemonStartMu.Lock()
	defer daemonStartMu.Unlock()
	probe := &connBase{}
	defer probe.closeLocked()
	var deadline time.Time
	started := false
	var logPath string
	for {
		if err := _dial(probe); err == nil {
			return nil
		} else if errors.Is(err, errDaemonIncompatible) {
			return err
		}
		if started {
			if time.Now().After(deadline) {
				return fmt.Errorf("prime daemon did not start; see %s", logPath)
			}
		} else {
			// actually start the daemon
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			logDir := filepath.Join(home, ".ahweb")
			if err := os.MkdirAll(logDir, 0700); err != nil {
				return fmt.Errorf("could not create ahweb config directory: %w", err)
			}
			logPath = filepath.Join(logDir, "prime-daemon.log")
			logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				return fmt.Errorf("could not open prime daemon log: %w", err)
			}
			defer logFile.Close()
			cmd := exec.Command("nohup", "prime-agent", "--mode", "daemon", "--daemon-socket", _socketPath())
			if cwd, err := os.UserHomeDir(); err == nil {
				cmd.Dir = cwd
			}
			cmd.Stdout, cmd.Stderr = logFile, logFile
			cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
			if err := cmd.Start(); err != nil {
				return fmt.Errorf("could not start prime daemon: %w", err)
			}
			_ = cmd.Process.Release()
			started, deadline = true, time.Now().Add(15*time.Second)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	cmd := exec.Command("prime-agent", "--mode", "rpc", "--no-session")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open prime input: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open prime output: %w", err)
	}
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start prime: %w", err)
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
		return nil, fmt.Errorf("could not ask prime for models: %w", err)
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
			return nil, fmt.Errorf("prime refused model list: %s", response.Error)
		}
		models := make(map[string]harness.Model, len(response.Data.Models))
		for _, model := range response.Data.Models {
			modelID := model.Provider + "/" + model.ID
			entry := harness.Model{Harness: "prime", ID: modelID}
			if model.Reasoning {
				for _, level := range []string{"minimal", "low", "medium", "high", "xhigh", "max"} {
					if effort, ok := model.ThinkingLevelMap[level]; ok && effort != nil {
						entry.Reasoning = append(entry.Reasoning, level)
					}
				}
			}
			models[modelID] = entry
		}
		return models, nil
	}
	return nil, errors.New("prime closed before sending model list")
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	_samePath := func(a, b string) bool {
		resolve := func(value string) string {
			if abs, err := filepath.Abs(value); err == nil {
				value = abs
			}
			if real, err := filepath.EvalSymlinks(value); err == nil {
				return real
			}
			return filepath.Clean(value)
		}
		return resolve(a) == resolve(b)
	}
	_messageText := func(raw json.RawMessage) string {
		var text string
		if json.Unmarshal(raw, &text) == nil {
			return text
		}
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal(raw, &parts) == nil {
			for _, part := range parts {
				if part.Type == "text" {
					text += part.Text
				}
			}
		}
		return text
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find prime home: %w", err)
	}
	dir := filepath.Join(home, ".prime", "agent", "sessions")
	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("could not read prime sessions: %w", err)
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
		if !scanner.Scan() || json.Unmarshal(scanner.Bytes(), &header) != nil || header.Type != "session" || header.ID == "" || header.CWD == "" || !_samePath(header.CWD, realFolder) {
			_ = file.Close()
			continue
		}

		// actually read file + populate lite
		s := harness.SessionLite{ID: header.ID, RealPath: realFolder, Harness: "prime"}
		if parent, err := os.Open(header.ParentSession); err == nil {
			var parentHeader struct {
				ID string `json:"id"`
			}
			if scanner := bufio.NewScanner(parent); scanner.Scan() && json.Unmarshal(scanner.Bytes(), &parentHeader) == nil {
				s.ParentSessID = parentHeader.ID
			}
			_ = parent.Close()
		}
		var firstMessage string
		gotFirstMessage, hasMessage := false, false
		for scanner.Scan() {
			var entry struct {
				Type    string `json:"type"`
				Message struct {
					Role      string          `json:"role"`
					Content   json.RawMessage `json:"content"`
					Timestamp json.RawMessage `json:"timestamp"`
				} `json:"message"`
			}
			if json.Unmarshal(scanner.Bytes(), &entry) != nil {
				continue
			}
			if entry.Type == "message" {
				hasMessage = true
			}
			if entry.Type == "message" && entry.Message.Role == "user" && !gotFirstMessage {
				gotFirstMessage = true
				firstMessage = _messageText(entry.Message.Content)
			}
			if entry.Type == "message" {
				if timestamp := messageTimestamp(entry.Message.Timestamp); timestamp > s.LastTimestampMs {
					s.LastTimestampMs = timestamp
				}
			}
		}
		if err := scanner.Err(); err != nil {
			_ = file.Close()
			continue
		}
		s.FirstUserMsg = firstMessage
		if hasMessage {
			// we explicitly hide empty sessions
			sessions = append(sessions, s)
		}
		_ = file.Close()
	}
	return sessions, nil
}

func (c *SessionConn) call(command string, params map[string]any, data any) error {
	return connCmd0.call(c.ActiveSessionID, command, params, data)
}

// === session admin fns

func (*Adapter) CleanUp(s *harness.Session) error {
	sessionConnsMu.Lock()
	conn := sessionConns[s.RealPath][s.ID]
	sessionConnsMu.Unlock()
	if conn == nil {
		return nil
	}

	conn.Mu.Lock()
	if conn.ActiveSessionID != "" {
		unsub(conn)
		var state struct {
			AttachedClients int `json:"attachedClients"`
		}
		if err := conn.call("get_state", nil, &state); err == nil && state.AttachedClients == 0 {
			_ = conn.call("kill", nil, nil)
		}
	}
	conn.ActiveSessionID, conn.MsgCursor = "", 0
	conn.EventSequence, conn.EventGeneration = 0, ""
	conn.Mu.Unlock()

	sessionConnsMu.Lock()
	if sessions := sessionConns[s.RealPath]; sessions[s.ID] == conn {
		delete(sessions, s.ID)
		if len(sessions) == 0 {
			delete(sessionConns, s.RealPath)
		}
	}
	sessionConnsMu.Unlock()
	return nil
}

func (a *Adapter) Connect(s *harness.Session, force bool) error {
	// setting up all the cache and structs
	sessionConnsMu.Lock()
	if sessionConns[s.RealPath] == nil {
		sessionConns[s.RealPath] = make(map[string]*SessionConn)
	}
	conn := sessionConns[s.RealPath][s.ID]
	if conn == nil {
		conn = &SessionConn{}
		sessionConns[s.RealPath][s.ID] = conn
	}
	sessionConnsMu.Unlock()
	conn.Mu.Lock()
	defer conn.Mu.Unlock()

	if !connCmd0.isConnected() {
		if err := a.Ready(); err != nil {
			return err
		}
		if err := connCmd0.Ready(); err != nil {
			return err
		}
	}
	if conn.ActiveSessionID != "" {
		var state struct {
			IsStreaming bool `json:"isStreaming"`
		}
		if err := conn.call("get_state", nil, &state); err == nil {
			wasListening := connEvt0.isSub(conn.ActiveSessionID)
			_ = sub(conn)
			if force || !wasListening {
				if !wasListening {
					conn.EventSequence, conn.EventGeneration = 0, ""
				}
				s.Mu.Lock()
				s.DoFullHistory = true
				s.Mu.Unlock()
			}
			return nil
		}
		unsub(conn)
		conn.ActiveSessionID, conn.MsgCursor = "", 0
		conn.EventSequence, conn.EventGeneration = 0, ""
	}

	// create session jsonl if missing, cos prime agent daemon needs that.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find prime home: %w", err)
	}
	path := filepath.Join(home, ".prime", "agent", "sessions", s.ID+".jsonl")
	if _, err := os.Stat(path); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not inspect prime session: %w", err)
		}
		header := struct {
			Type      string `json:"type"`
			Version   int    `json:"version"`
			ID        string `json:"id"`
			Timestamp string `json:"timestamp"`
			CWD       string `json:"cwd"`
			RLMDepth  int    `json:"rlmDepth"`
		}{"session", 3, s.ID, time.Now().UTC().Format("2006-01-02T15:04:05.000Z"), s.RealPath, 0}
		raw, err := json.Marshal(header)
		if err != nil {
			return fmt.Errorf("could not encode prime session header: %w", err)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return fmt.Errorf("could not create prime sessions directory: %w", err)
		}
		if err := os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
			return fmt.Errorf("could not create prime session: %w", err)
		}
	}

	if err := connCmd0.Ready(); err != nil {
		if readyErr := a.Ready(); readyErr != nil {
			return readyErr
		}
		if err := connCmd0.Ready(); err != nil {
			return err
		}
	}

	// checked: the create cmd to the daemon works even for multi connection
	var created struct {
		ActiveSessionID string `json:"activeSessionId"`
		SessionID       string `json:"sessionId"`
	}
	if err := conn.call("create", map[string]any{
		"sessionPath": path,
		"config":      map[string]any{"cwd": s.RealPath},
	}, &created); err != nil {
		message := err.Error()
		if strings.Contains(message, "Unknown daemon command") || strings.Contains(message, "malformed") {
			return fmt.Errorf("%w: %v", errDaemonUnavailable, err)
		}
		return fmt.Errorf("prime daemon could not create session: %w", err)
	}
	if created.SessionID != "" && created.SessionID != s.ID {
		return fmt.Errorf("prime daemon created unexpected session %s", created.SessionID)
	}
	if created.ActiveSessionID == "" {
		return errors.New("prime daemon returned no active session")
	}
	conn.ActiveSessionID, conn.MsgCursor = created.ActiveSessionID, 0
	conn.EventSequence, conn.EventGeneration = 0, ""
	_ = sub(conn)
	s.Mu.Lock()
	s.DoFullHistory = true
	s.Mu.Unlock()
	return nil
}

func (*Adapter) CanDelete() bool { return true }

func (a *Adapter) Delete(s *harness.Session) error {
	if s.GetIsBusy() {
		return errors.New("cannot delete a busy Prime session")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	sessionPath := filepath.Join(home, ".prime", "agent", "sessions", s.ID+".jsonl")
	file, err := os.Open(sessionPath)
	if err != nil {
		return err
	}
	originalInfo, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return err
	}
	var header struct {
		ID string `json:"id"`
	}
	err = json.NewDecoder(bufio.NewReader(file)).Decode(&header)
	_ = file.Close()
	if err != nil || header.ID != s.ID {
		return fmt.Errorf("Prime session %q was not found", s.ID)
	}
	entries, err := os.ReadDir(filepath.Dir(sessionPath))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" || entry.Name() == filepath.Base(sessionPath) {
			continue
		}
		child, err := os.Open(filepath.Join(filepath.Dir(sessionPath), entry.Name()))
		if err != nil {
			continue
		}
		var childHeader struct {
			ParentSession string `json:"parentSession"`
		}
		err = json.NewDecoder(bufio.NewReader(child)).Decode(&childHeader)
		_ = child.Close()
		if err == nil && childHeader.ParentSession == sessionPath {
			return errors.New("cannot delete a Prime session with child sessions")
		}
	}
	if err := a.Connect(s, false); err != nil {
		return err
	}
	currentInfo, err := os.Stat(sessionPath)
	if err != nil || !os.SameFile(originalInfo, currentInfo) {
		return fmt.Errorf("Prime session %q changed while opening it", s.ID)
	}
	conn, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	conn.Mu.Lock()
	var state struct {
		IsStreaming     bool `json:"isStreaming"`
		AttachedClients int  `json:"attachedClients"`
	}
	err = conn.call("get_state", nil, &state)
	conn.Mu.Unlock()
	if err != nil {
		return err
	}
	if state.IsStreaming || state.AttachedClients > 0 {
		return errors.New("cannot delete a Prime session while another client is using it")
	}
	if err := a.CleanUp(s); err != nil {
		return err
	}
	return os.Remove(sessionPath)
}

// === session fns

type primeTree struct {
	LeafID    *string `json:"leafId"`
	FlatNodes []struct {
		Entry struct {
			ID       string  `json:"id"`
			ParentID *string `json:"parentId"`
			Type     string  `json:"type"`
			Message  struct {
				Role      string          `json:"role"`
				Content   json.RawMessage `json:"content"`
				Timestamp json.RawMessage `json:"timestamp"`
			} `json:"message"`
		} `json:"entry"`
	} `json:"flatNodes"`
}

func (a *Adapter) Rewind(s *harness.Session, rewindID string) error {
	conn, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	var state struct {
		IsStreaming  bool `json:"isStreaming"`
		IsCompacting bool `json:"isCompacting"`
	}
	if err := conn.call("get_state", nil, &state); err != nil {
		return err
	}
	if state.IsStreaming || state.IsCompacting {
		return errors.New("Prime session is busy")
	}
	var tree primeTree
	if err := conn.call("get_session_tree", nil, &tree); err != nil {
		return err
	}

	// finding the node-1
	var parent *string
	found := false
	for _, node := range tree.FlatNodes {
		if node.Entry.ID == rewindID && node.Entry.Type == "message" && node.Entry.Message.Role == "user" {
			parent, found = node.Entry.ParentID, true
			break
		}
	}
	if !found {
		return errors.New("Prime rewind target was not found")
	}
	var result struct {
		Cancelled bool `json:"cancelled"`
	}
	if err := conn.call("navigate_tree", map[string]any{"targetId": rewindID, "summarize": false}, &result); err != nil {
		return err
	}
	if result.Cancelled {
		return errors.New("Prime rewind was cancelled")
	}
	if err := conn.call("get_session_tree", nil, &tree); err != nil {
		return err
	}
	if (parent == nil) != (tree.LeafID == nil) || parent != nil && *parent != *tree.LeafID {
		return errors.New("Prime rewind did not reach the requested parent")
	}
	conn.MsgCursor = 0
	s.Mu.Lock()
	s.DoFullHistory = true
	s.Mu.Unlock()
	return nil
}

func (a *Adapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	model, reasoning, prompt := s.PromptModel, s.PromptReasoning, s.SendingMsg
	images := append([]string(nil), s.SendingImages...)
	s.Mu.Unlock()
	sessionConnsMu.Lock()
	conn := sessionConns[s.RealPath][s.ID]
	sessionConnsMu.Unlock()
	if conn == nil {
		return errors.New("prime session is not connected")
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.ActiveSessionID == "" || !connCmd0.isConnected() {
		return errors.New("prime session is not connected")
	}

	// setting the models
	if model != nil {
		provider, modelID, ok := strings.Cut(model.ID, "/")
		if !ok || provider == "" || modelID == "" {
			return fmt.Errorf("invalid prime model ID: %q", model.ID)
		}
		if err := conn.call("set_model", map[string]any{"provider": provider, "modelId": modelID}, nil); err != nil {
			return err
		}
		if reasoning == "" && len(model.Reasoning) > 0 {
			reasoning = model.Reasoning[0]
		}
		if reasoning != "" {
			if err := conn.call("set_thinking_level", map[string]any{"level": reasoning}, nil); err != nil {
				return err
			}
		}
	}

	// forming the payload
	payload := map[string]any{"message": prompt}
	if len(images) > 0 {
		parts := make([]harness.Image, 0, len(images))
		for _, uri := range images {
			part, err := harness.ParseImage(uri)
			if err != nil {
				return err
			}
			parts = append(parts, part)
		}
		payload["images"] = parts
	}
	if err := conn.call("prompt", payload, nil); err != nil {
		return err
	}
	s.Mu.Lock()
	s.BusyCli = true
	s.Mu.Unlock()
	return nil
}

func (a *Adapter) Stop(s *harness.Session) error {
	conn, err := a.sessionConn(s)
	if err != nil {
		return nil
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.ActiveSessionID == "" || !connCmd0.isConnected() {
		return nil
	}
	err = conn.call("abort", nil, nil)
	if err != nil && (strings.Contains(err.Error(), "Unknown active session") || strings.Contains(err.Error(), "not processing")) {
		return nil
	}
	return err
}

func (a *Adapter) Poll(s *harness.Session) error {
	conn, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	conn.Mu.Lock()
	defer conn.Mu.Unlock()
	if conn.ActiveSessionID == "" || !connCmd0.isConnected() {
		return errors.New("prime session is not connected")
	}

	s.Mu.Lock()
	full := s.DoFullHistory
	s.Mu.Unlock()
	if !connEvt0.isSub(conn.ActiveSessionID) {
		_ = sub(conn)
		full = true
		conn.EventSequence, conn.EventGeneration = 0, ""
		s.Mu.Lock()
		s.DoFullHistory = true
		s.Mu.Unlock()
	}
	updates, needsFull := connEvt0.drainSub(conn.ActiveSessionID)
	full = full || needsFull

	type primeMessage struct {
		Role       string          `json:"role"`
		Content    json.RawMessage `json:"content"`
		Timestamp  json.RawMessage `json:"timestamp"`
		Model      string          `json:"model"`
		ToolCallID string          `json:"toolCallId"`
		ToolName   string          `json:"toolName"`
		IsError    bool            `json:"isError"`
	}
	var messages []primeMessage
	ids := []string(nil)

	rebuild := full
	refreshMessages := full || !connEvt0.isSub(conn.ActiveSessionID)
	messageCountDelta := 0
	nextEventSequence, nextEventGeneration := conn.EventSequence, conn.EventGeneration

	// partial poll
	for _, update := range updates {
		var frame struct {
			Type  string `json:"type"`
			Event struct {
				Type    string          `json:"type"`
				Message json.RawMessage `json:"message"`
			} `json:"event"`
			Meta struct {
				Sequence uint64 `json:"sequence"`
				Cursor   struct {
					Generation string `json:"generation"`
					Sequence   uint64 `json:"sequence"`
				} `json:"cursor"`
			} `json:"meta"`
		}
		if err := json.Unmarshal(update, &frame); err != nil || frame.Type != "session_event" {
			refreshMessages, rebuild = true, true
			break
		}
		sequence := frame.Meta.Sequence
		if sequence == 0 {
			sequence = frame.Meta.Cursor.Sequence
		}
		generation := frame.Meta.Cursor.Generation
		if sequence == 0 || nextEventSequence > 0 && (generation != "" && nextEventGeneration != "" && generation != nextEventGeneration || sequence != nextEventSequence+1) {
			refreshMessages, rebuild = true, true
		}
		if sequence > 0 {
			nextEventSequence = sequence
		}
		if generation != "" {
			nextEventGeneration = generation
		}
		switch frame.Event.Type {
		case "session_replaced", "session_resynced", "session_closed":
			refreshMessages, rebuild = true, true
		}
		if !refreshMessages && frame.Event.Type == "message_end" {
			var message primeMessage
			if err := json.Unmarshal(frame.Event.Message, &message); err != nil || message.Role == "" {
				refreshMessages, rebuild = true, true
				messages = nil
				messageCountDelta = 0
				continue
			}
			messages = append(messages, message)
			messageCountDelta++
		}
	}
	hasUserMessage := false
	for _, message := range messages {
		hasUserMessage = hasUserMessage || message.Role == "user"
	}

	// this is the full poll
	if refreshMessages {
		var messagesResponse struct {
			Messages []primeMessage `json:"messages"`
		}
		if err := conn.call("get_messages", nil, &messagesResponse); err != nil {
			connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
			return err
		}
		messages = messagesResponse.Messages
		var tree primeTree
		if err := conn.call("get_session_tree", nil, &tree); err != nil {
			connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
			return err
		}
		byID := make(map[string]int, len(tree.FlatNodes))
		for i, node := range tree.FlatNodes {
			byID[node.Entry.ID] = i
		}
		var branch []string
		for id := tree.LeafID; id != nil; {
			i, ok := byID[*id]
			if !ok {
				break
			}
			entry := tree.FlatNodes[i].Entry
			if entry.Type == "message" {
				branch = append(branch, entry.ID)
			}
			id = entry.ParentID
		}
		ids = make([]string, len(messages))
		if len(branch) == len(messages) {
			matched := true
			for i, id := range branch {
				j := len(branch) - 1 - i
				if tree.FlatNodes[byID[id]].Entry.Message.Role != messages[j].Role {
					matched = false
					break
				}
				ids[j] = id
			}
			if !matched {
				clear(ids)
			}
		}
		rebuild = rebuild || len(messages) < conn.MsgCursor
		messageCountDelta = 0
	} else if hasUserMessage {
		// we need to find rewindID (therefore poll whole tree)
		var tree primeTree
		if err := conn.call("get_session_tree", nil, &tree); err != nil {
			connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
			return err
		}
		byID := make(map[string]int, len(tree.FlatNodes))
		for i, node := range tree.FlatNodes {
			byID[node.Entry.ID] = i
		}
		ids = make([]string, len(messages))
		_messageKey := func(role string, content, timestamp json.RawMessage) string {
			var normalizedContent, normalizedTimestamp any
			_ = json.Unmarshal(content, &normalizedContent)
			_ = json.Unmarshal(timestamp, &normalizedTimestamp)
			key, _ := json.Marshal(struct {
				Role      string `json:"role"`
				Content   any    `json:"content"`
				Timestamp any    `json:"timestamp"`
			}{role, normalizedContent, normalizedTimestamp})
			return string(key)
		}
		type treeUser struct{ id, key string }
		var branchUsers []treeUser
		for id := tree.LeafID; id != nil; {
			i, ok := byID[*id]
			if !ok {
				break
			}
			entry := tree.FlatNodes[i].Entry
			if entry.Type == "message" && entry.Message.Role == "user" {
				branchUsers = append(branchUsers, treeUser{entry.ID, _messageKey(entry.Message.Role, entry.Message.Content, entry.Message.Timestamp)})
			}
			id = entry.ParentID
		}
		userMessages := 0
		for _, message := range messages {
			if message.Role == "user" {
				userMessages++
			}
		}
		if len(branchUsers) < userMessages {
			connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
			return nil
		}
		nextBranchUser := 0
		for i := len(messages) - 1; i >= 0; i-- {
			message := messages[i]
			if message.Role != "user" || len(branchUsers) < userMessages || nextBranchUser >= len(branchUsers) {
				continue
			}
			key := _messageKey(message.Role, message.Content, message.Timestamp)
			match := nextBranchUser
			for j := nextBranchUser; j < len(branchUsers); j++ {
				if branchUsers[j].key == key {
					match = j
					break
				}
			}
			ids[i] = branchUsers[match].id
			nextBranchUser = match + 1
		}
	}
	start := 0
	if refreshMessages {
		start = conn.MsgCursor
	}
	if rebuild {
		start = 0
	}

	// build the Event objs
	events := make([]harness.Event, 0)
	for i := start; i < len(messages); i++ {
		message := messages[i]
		if message.Role == "custom" {
			continue
		}
		timestamp := messageTimestamp(message.Timestamp)
		var content string
		var images []string
		if json.Unmarshal(message.Content, &content) != nil {
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
			if json.Unmarshal(message.Content, &parts) == nil {
				for _, part := range parts {
					switch part.Type {
					case "text":
						content += part.Text
					case "image":
						if uri := harness.DataURI(part.MimeType, part.Data); uri != "" {
							images = append(images, uri)
						}
					case "thinking":
						if text := strings.TrimSpace(part.Thinking); text != "" && message.Role == "assistant" {
							events = append(events, harness.Event{Type: harness.EventThinking, Content: text, TimestampMs: timestamp, Model: message.Model})
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
		switch message.Role {
		case "toolResult":
			var resultParts []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			}
			var result string
			if json.Unmarshal(message.Content, &resultParts) == nil {
				for _, part := range resultParts {
					if part.Type == "text" {
						result += part.Text
					}
				}
			} else {
				_ = json.Unmarshal(message.Content, &result)
			}
			events = append(events, harness.Event{Type: harness.EventToolEnd, ToolName: message.ToolName, ToolCallID: message.ToolCallID, Content: strings.TrimSpace(result), Images: images, ToolFailed: message.IsError, TimestampMs: timestamp})
		case "user":
			if content = strings.TrimSpace(content); content != "" || len(images) > 0 {
				events = append(events, harness.Event{Type: harness.EventUser, Content: content, Images: images, TimestampMs: timestamp, RewindID: ids[i]})
			}
		case "assistant":
			if content = strings.TrimSpace(content); content != "" || len(images) > 0 {
				events = append(events, harness.Event{Type: harness.EventAssistant, Content: content, Images: images, TimestampMs: timestamp, Model: message.Model})
			}
		}
	}

	// update stats
	var stats struct {
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
	}
	if err := conn.call("get_session_stats", nil, &stats); err != nil {
		connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
		return err
	}
	var state struct {
		IsStreaming  bool `json:"isStreaming"`
		IsCompacting bool `json:"isCompacting"`
	}
	if err := conn.call("get_state", nil, &state); err != nil {
		connEvt0.undrainSub(conn.ActiveSessionID, updates, needsFull)
		return err
	}
	if refreshMessages {
		conn.MsgCursor = len(messages)
	} else {
		conn.MsgCursor += messageCountDelta
	}
	conn.EventSequence, conn.EventGeneration = nextEventSequence, nextEventGeneration

	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.BusyCli = state.IsStreaming || state.IsCompacting
	if rebuild {
		s.Events = events
	} else {
		s.Events = append(s.Events, events...)
	}
	if rebuild || s.FirstUserMsg == "" {
		s.FirstUserMsg = ""
		for _, event := range events {
			if event.Type == harness.EventUser {
				s.FirstUserMsg = event.Content
				break
			}
		}
	}
	s.FullInputToken = stats.Tokens.Input
	s.FullOutputToken = stats.Tokens.Output
	s.CacheReadToken = stats.Tokens.CacheRead
	s.CacheWriteToken = stats.Tokens.CacheWrite
	s.FullCost = stats.Cost
	s.ContextTokens = 0
	if stats.ContextUsage.Tokens != nil {
		s.ContextTokens = *stats.ContextUsage.Tokens
	}
	s.ContextWindow = stats.ContextUsage.ContextWindow
	for _, message := range messages {
		if timestamp := messageTimestamp(message.Timestamp); timestamp > s.LastTimestampMs {
			s.LastTimestampMs = timestamp
		}
	}
	s.DoFullHistory = false
	return nil
}

// === utils fns

func (*Adapter) sessionConn(s *harness.Session) (*SessionConn, error) {
	sessionConnsMu.Lock()
	defer sessionConnsMu.Unlock()
	if conn := sessionConns[s.RealPath][s.ID]; conn != nil {
		return conn, nil
	}
	return nil, errors.New("prime session is not connected")
}

func messageTimestamp(raw json.RawMessage) int64 {
	// prime use both unix ms and ISO-8601 for timestamp (in custom_message)
	var ms int64
	if json.Unmarshal(raw, &ms) == nil {
		return ms
	}
	var iso string
	if json.Unmarshal(raw, &iso) == nil {
		if parsed, err := time.Parse(time.RFC3339Nano, iso); err == nil {
			return parsed.UnixMilli()
		}
	}
	return 0
}
