package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type tokenUsage struct {
	Total struct {
		InputTokens           int64 `json:"inputTokens"`
		OutputTokens          int64 `json:"outputTokens"`
		CachedInputTokens     int64 `json:"cachedInputTokens"`
		CacheWriteInputTokens int64 `json:"cacheWriteInputTokens"`
		TotalTokens           int64 `json:"totalTokens"`
	} `json:"total"`
	Last struct {
		TotalTokens int64 `json:"totalTokens"`
	} `json:"last"`
	ModelContextWindow int64 `json:"modelContextWindow"`
}

// === adapter admin fns

type Adapter struct{ harness.DefaultAdapter }

type itemEventRef struct {
	indices   []int
	startedAt int64
	completed bool
}

type SessionConn struct {
	mu                 sync.Mutex
	harnessID          string
	codexTurnID        string
	isWriter           bool
	firstMetadataErrAt time.Time
	usage              tokenUsage
	errors             []harness.Event
	isLoaded           bool
	eventRefs          map[string]itemEventRef
	eventCache         []connEvent
}

var (
	cacheMu           sync.Mutex
	conns             = map[string]map[string]*SessionConn{}
	sessIDToHarnessID = map[string]map[string]string{}
	harnessIDToSessID = map[string]map[string]string{}
	harnessIDToPrompt = map[string]map[string]string{}
)

func (*Adapter) Ready() error { return ensureRPC() }

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	if err := ensureRPC(); err != nil {
		return nil, err
	}
	models := map[string]harness.Model{}
	cursor := ""
	for {
		params := map[string]any{"limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, _, err := connRPC0.call("model/list", params)
		if err != nil {
			return nil, err
		}
		var page struct {
			Data []struct {
				Model     string `json:"model"`
				Supported []struct {
					Reasoning string `json:"reasoningEffort"`
				} `json:"supportedReasoningEfforts"`
			} `json:"data"`
			NextCursor *string `json:"nextCursor"`
		}
		if err := decode(raw, &page); err != nil {
			return nil, err
		}
		for _, item := range page.Data {
			if item.Model == "" {
				continue
			}
			model := harness.Model{Harness: "codex", ID: "codex/" + item.Model}
			for _, effort := range item.Supported {
				if effort.Reasoning != "" {
					model.Reasoning = append(model.Reasoning, effort.Reasoning)
				}
			}
			models[model.ID] = model
		}
		if page.NextCursor == nil || *page.NextCursor == "" {
			break
		}
		cursor = *page.NextCursor
	}
	return models, nil
}

func (*Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	if err := ensureRPC(); err != nil {
		return nil, err
	}

	type harnessSession struct {
		ID            string          `json:"id"`
		Preview       string          `json:"preview"`
		Name          string          `json:"name"`
		AgentNickname string          `json:"agentNickname"`
		UpdatedAt     int64           `json:"updatedAt"`
		ParentID      string          `json:"parentThreadId"`
		Source        json.RawMessage `json:"source"`
	}
	_listSessions := func(baseParams map[string]any) ([]harnessSession, error) {
		var sessions []harnessSession
		cursor := ""
		for {
			params := make(map[string]any, len(baseParams)+1)
			for k, v := range baseParams {
				params[k] = v
			}
			if cursor != "" {
				params["cursor"] = cursor
			}
			raw, _, err := connRPC0.call("thread/list", params)
			if err != nil {
				return nil, err
			}
			var result struct {
				Data       []harnessSession `json:"data"`
				NextCursor *string          `json:"nextCursor"`
			}
			if err := decode(raw, &result); err != nil {
				return nil, err
			}
			sessions = append(sessions, result.Data...)
			if result.NextCursor == nil || *result.NextCursor == "" {
				break
			}
			cursor = *result.NextCursor
		}
		return sessions, nil
	}
	_toSessionLite := func(hs harnessSession) harness.SessionLite {
		var source struct {
			SubAgent struct {
				ThreadSpawn struct {
					AgentPath      string `json:"agent_path"`
					AgentNickname  string `json:"agent_nickname"`
					ParentThreadID string `json:"parent_thread_id"`
				} `json:"thread_spawn"`
			} `json:"subAgent"`
		}
		_ = json.Unmarshal(hs.Source, &source)
		spawn := source.SubAgent.ThreadSpawn

		id := hs.ID
		if local := harnessIDToSessID[realFolder][hs.ID]; local != "" {
			id = local
		}
		parentID := hs.ParentID
		if parentID == "" {
			parentID = spawn.ParentThreadID
		}
		if local := harnessIDToSessID[realFolder][parentID]; local != "" {
			parentID = local
		}

		firstUserMsg := strings.TrimSpace(hs.Preview)
		if firstUserMsg == "" {
			firstUserMsg = strings.TrimSpace(hs.Name)
		}
		if firstUserMsg == "" {
			agentPath := strings.TrimSpace(spawn.AgentPath)
			nickname := strings.TrimSpace(hs.AgentNickname)
			if nickname == "" {
				nickname = strings.TrimSpace(spawn.AgentNickname)
			}
			switch {
			case agentPath != "" && nickname != "":
				firstUserMsg = agentPath + " (" + nickname + ")"
			case agentPath != "":
				firstUserMsg = agentPath
			case nickname != "":
				firstUserMsg = nickname
			}
		}
		return harness.SessionLite{
			ID: id, RealPath: realFolder, Harness: "codex",
			FirstUserMsg: firstUserMsg, LastTimestampMs: hs.UpdatedAt * 1000,
			ParentSessID: parentID,
		}
	}

	topLevel, err := _listSessions(map[string]any{"cwd": realFolder, "limit": 100, "sortKey": "updated_at", "sortDirection": "desc"})
	if err != nil {
		return nil, err
	}

	var sessions []harness.SessionLite
	seen := make(map[string]bool, len(topLevel))
	rootIDs := make([]string, 0, len(topLevel))
	for _, hs := range topLevel {
		rootIDs = append(rootIDs, hs.ID)
		if seen[hs.ID] {
			continue
		}
		seen[hs.ID] = true
		sessions = append(sessions, _toSessionLite(hs))
	}
	for harnessID := range harnessIDToSessID[realFolder] {
		rootIDs = append(rootIDs, harnessID)
	}
	rootSeen := make(map[string]bool, len(rootIDs))
	for _, rootID := range rootIDs {
		if rootSeen[rootID] {
			continue
		}
		rootSeen[rootID] = true
		descendants, err := _listSessions(map[string]any{"ancestorThreadId": rootID, "limit": 100, "sortKey": "updated_at", "sortDirection": "desc"})
		if err != nil {
			return nil, err
		}
		for _, hs := range descendants {
			if seen[hs.ID] {
				continue
			}
			seen[hs.ID] = true
			sessions = append(sessions, _toSessionLite(hs))
		}
	}
	return sessions, nil
}

// === session admin fns

func (a *Adapter) Connect(s *harness.Session, _ bool) error {
	s.Mu.Lock()
	id := s.ID
	s.Mu.Unlock()

	// its kinda like a hacky to tell if the id is localID or codexID
	isLocal := len(id) == 32
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			isLocal = false
			break
		}
	}

	// early return if possible
	cacheMu.Lock()
	cached := conns[s.RealPath][id]
	threadID := id
	if isLocal {
		threadID = sessIDToHarnessID[s.RealPath][id]
	}
	cacheMu.Unlock()
	// History refreshes must not resume a healthy, possibly unsaved thread.
	if cached != nil && connRPC0 != nil && connRPC0.chkConnection() == nil {
		connRPC0.mu.Lock()
		registered := connRPC0.threads[threadID] != nil
		connRPC0.mu.Unlock()
		if registered {
			return nil
		}
	}

	// startup the connection
	_ = a.CleanUp(s)
	if err := ensureRPC(); err != nil {
		return err
	}
	c := &SessionConn{eventRefs: map[string]itemEventRef{}}
	method := "thread/start"
	params := map[string]any{"cwd": s.RealPath, "approvalPolicy": "never", "sandbox": "danger-full-access"}
	if threadID != "" {
		method = "thread/read"
		params = map[string]any{"threadId": threadID}
	}
	raw, _, err := connRPC0.call(method, params)
	if err != nil {
		return err
	}

	// init all the cache and pointer maps
	var result struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := decode(raw, &result); err != nil {
		return err
	}
	if result.Thread.ID == "" {
		return errors.New("Codex returned no thread ID")
	}
	c.mu.Lock()
	c.harnessID = result.Thread.ID
	c.isWriter = method == "thread/start"
	c.mu.Unlock()
	connRPC0.mu.Lock()
	connRPC0.threads[result.Thread.ID] = c
	connRPC0.mu.Unlock()
	cacheMu.Lock()

	if isLocal {
		if sessIDToHarnessID[s.RealPath] == nil {
			sessIDToHarnessID[s.RealPath] = map[string]string{}
		}
		if harnessIDToSessID[s.RealPath] == nil {
			harnessIDToSessID[s.RealPath] = map[string]string{}
		}
		sessIDToHarnessID[s.RealPath][id] = result.Thread.ID
		harnessIDToSessID[s.RealPath][result.Thread.ID] = id
	}
	if conns[s.RealPath] == nil {
		conns[s.RealPath] = map[string]*SessionConn{}
	}
	conns[s.RealPath][id] = c
	cacheMu.Unlock()
	return nil
}

func (a *Adapter) CleanUp(s *harness.Session) error {
	cacheMu.Lock()
	s.Mu.Lock()
	id := s.ID
	s.Mu.Unlock()
	c := conns[s.RealPath][id]
	cacheMu.Unlock()
	if c != nil {
		c.mu.Lock()
		threadID, isWriter := c.harnessID, c.isWriter
		c.mu.Unlock()

		// if its writer, we actually unsub to try to give back control to other clients
		if connRPC0 != nil {
			if isWriter && connRPC0.chkConnection() == nil {
				_, _, err := connRPC0.call("thread/unsubscribe", map[string]any{"threadId": threadID})
				if err != nil {
					return err
				}
			}
			connRPC0.mu.Lock()
			delete(connRPC0.threads, threadID)
			connRPC0.mu.Unlock()
		}

		c.mu.Lock()
		c.isWriter = false
		c.mu.Unlock()
		cacheMu.Lock()

		// clearing all the cache
		if conns[s.RealPath][id] == c {
			delete(conns[s.RealPath], id)
			if len(conns[s.RealPath]) == 0 {
				delete(conns, s.RealPath)
			}
		}
		cacheMu.Unlock()
	}
	return nil
}

func (*Adapter) CanDelete() bool { return true }

func (a *Adapter) Delete(s *harness.Session) error {
	if s.GetIsBusy() {
		return errors.New("cannot delete a busy Codex session")
	}
	cacheMu.Lock()
	threadID := sessIDToHarnessID[s.RealPath][s.ID]
	c := conns[s.RealPath][s.ID]
	cacheMu.Unlock()
	if c != nil {
		c.mu.Lock()
		if c.harnessID != "" {
			threadID = c.harnessID
		}
		c.mu.Unlock()
	}
	if threadID == "" {
		threadID = s.ID
		if len(s.ID) == 32 {
			return fmt.Errorf("Codex native thread ID for %q is unavailable", s.ID)
		}
	}
	if err := a.CleanUp(s); err != nil {
		return err
	}
	_, _, err := connRPC0.call("thread/delete", map[string]any{"threadId": threadID})
	if err != nil {
		return err
	}
	cacheMu.Lock()
	delete(sessIDToHarnessID[s.RealPath], s.ID)
	delete(harnessIDToSessID[s.RealPath], threadID)
	delete(harnessIDToPrompt[s.RealPath], threadID)
	delete(harnessIDToPrompt[s.RealPath], s.ID)
	cacheMu.Unlock()
	return nil
}

func (a *Adapter) Rewind(s *harness.Session, rewindID string) error {
	c, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	c.mu.Lock()
	threadID, isWriter := c.harnessID, c.isWriter
	c.mu.Unlock()
	if !isWriter {
		if _, _, err := connRPC0.call("thread/resume", map[string]any{"threadId": threadID, "cwd": s.RealPath, "approvalPolicy": "never", "sandbox": "danger-full-access"}); err != nil {
			return err
		}
		c.mu.Lock()
		c.isWriter = true
		c.mu.Unlock()
	}
	_, _, err = connRPC0.call("thread/revert", map[string]any{"threadId": threadID, "beforeTurnId": rewindID})
	return err
}

// === session fns

func (a *Adapter) SendPrompt(s *harness.Session) error {
	c, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	s.Mu.Lock()
	prompt, imgs, model, effort := s.SendingMsg, append([]string(nil), s.SendingImages...), s.PromptModel, s.PromptReasoning
	s.Mu.Unlock()

	input := make([]any, 0, len(imgs)+1)
	if prompt != "" {
		input = append(input, map[string]any{"type": "text", "text": prompt})
	}
	for _, uri := range imgs {
		if _, err := harness.ParseImage(uri); err != nil {
			return err
		}
		input = append(input, map[string]any{"type": "image", "url": uri})
	}
	c.mu.Lock()
	threadID, isWriter := c.harnessID, c.isWriter
	c.mu.Unlock()

	// swap the connection to write (if needed)
	if !isWriter {
		if _, _, err := connRPC0.call("thread/resume", map[string]any{"threadId": threadID, "cwd": s.RealPath, "approvalPolicy": "never", "sandbox": "danger-full-access"}); err != nil {
			return err
		}
		c.mu.Lock()
		c.isWriter = true
		c.mu.Unlock()
	}

	// form the args and send it out
	params := map[string]any{"threadId": threadID, "input": input, "cwd": s.RealPath}
	if model != nil {
		params["model"] = strings.TrimPrefix(model.ID, "codex/")
	}
	if effort != "" {
		params["effort"] = effort
	}
	raw, _, err := connRPC0.call("turn/start", params)
	if err != nil {
		return err
	}
	var result struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := decode(raw, &result); err != nil {
		return err
	}

	// setting all the updated fields
	c.mu.Lock()
	c.codexTurnID, c.errors = result.Turn.ID, nil
	c.mu.Unlock()
	s.Mu.Lock()
	s.BusyCli = true
	s.Mu.Unlock()
	return nil
}

func (a *Adapter) Stop(s *harness.Session) error {
	c, err := a.sessionConn(s)
	if err != nil {
		return nil
	}
	c.mu.Lock()
	turn := c.codexTurnID
	c.mu.Unlock()
	if turn == "" {
		return nil
	}
	_, _, err = connRPC0.call("turn/interrupt", map[string]any{"threadId": c.harnessID, "turnId": turn})
	return err
}

func (a *Adapter) Poll(s *harness.Session) error {
	c, err := a.sessionConn(s)
	if err != nil {
		return err
	}
	updates := c.drain()
	committed := false
	defer func() {
		if !committed {
			c.undrain(updates)
		}
	}()

	_itemText := func(item map[string]any) string {
		if text, ok := item["text"].(string); ok {
			return text
		}
		if review, ok := item["review"].(string); ok {
			return review
		}
		keys := []string{"content"}
		if item["type"] == "reasoning" {
			keys = []string{"summary", "content"}
		}
		var out strings.Builder
		for _, key := range keys {
			switch parts := item[key].(type) {
			case []any:
				for _, part := range parts {
					switch v := part.(type) {
					case string:
						out.WriteString(v)
					case map[string]any:
						text, _ := v["text"].(string)
						out.WriteString(text)
					}
				}
			case string:
				out.WriteString(parts)
			}
			if out.Len() > 0 {
				break
			}
		}
		return out.String()
	}

	_itemJSON := func(item map[string]any, omitResult bool) string {
		fields := make(map[string]any, len(item))
		for key, value := range item {
			if key == "type" || key == "id" || key == "result" && omitResult {
				continue
			}
			fields[key] = value
		}
		if raw, err := json.MarshalIndent(fields, "", "  "); err == nil {
			return string(raw)
		}
		return _itemText(item)
	}

	_toolEvent := func(
		name, id, content, content0 string,
		timestamp int64,
		failed bool,
		images []string,
		complete bool,
	) []harness.Event {
		startContent := content0
		if startContent == "" {
			startContent = content
			if chars := []rune(startContent); len(chars) > 160 {
				startContent = string(chars[:160])
			}
		}
		events := []harness.Event{{
			Type: harness.EventToolStart, ToolName: name, ToolCallID: id,
			Content: startContent, TimestampMs: timestamp,
		}}
		if complete {
			events = append(events, harness.Event{
				Type: harness.EventToolEnd, ToolName: name, ToolCallID: id,
				ToolFailed: failed, Content: content, Images: images, TimestampMs: timestamp,
			})
		}
		return events
	}

	_childPrompt := func(item map[string]any) (string, []string) {
		if item["type"] != "collabAgentToolCall" {
			return "", nil
		}
		prompt, _ := item["prompt"].(string)
		prompt = strings.TrimSpace(prompt)
		var ids []string
		switch values := item["receiverThreadIds"].(type) {
		case []any:
			for _, value := range values {
				if id, ok := value.(string); ok && id != "" {
					ids = append(ids, id)
				}
			}
		case []string:
			ids = append(ids, values...)
		}
		return prompt, ids
	}

	_convert := func(item map[string]any, timestamp int64, turnID string) []harness.Event {
		typeName, _ := item["type"].(string)
		content := _itemText(item)
		switch typeName {
		case "userMessage":
			var images []string
			if parts, ok := item["content"].([]any); ok {
				for _, part := range parts {
					if image, ok := part.(map[string]any); ok && image["type"] == "image" {
						if url, ok := image["url"].(string); ok && strings.HasPrefix(url, "data:image/") {
							images = append(images, url)
						}
					}
				}
			}
			return []harness.Event{{Type: harness.EventUser, Content: content, Images: images, TimestampMs: timestamp, RewindID: turnID}}
		case "agentMessage":
			return []harness.Event{{Type: harness.EventAssistant, Content: content, TimestampMs: timestamp}}
		case "ContextCompaction", "contextCompaction", "compacted":
			return []harness.Event{{Type: harness.EventAssistant, Content: "Context compacted.", TimestampMs: timestamp}}
		case "reasoning":
			return []harness.Event{{Type: harness.EventThinking, Content: content, TimestampMs: timestamp}}
		case "commandExecution":
			id, _ := item["id"].(string)
			cmd, _ := item["command"].(string)
			status, _ := item["status"].(string)
			name := "command"
			if fields := strings.Fields(cmd); len(fields) > 0 {
				name = filepath.Base(fields[0])
			}
			content = _itemJSON(item, false)
			var content0 string
			if actions, ok := item["commandActions"]; ok {
				if raw, err := json.MarshalIndent(actions, "", "  "); err == nil {
					content0 = string(raw)
				}
			}
			return _toolEvent(name, id, content, content0, timestamp, status == "failed", nil, status != "inProgress")
		case "Extension", "extension", "fileChange", "mcpToolCall", "dynamicToolCall", "webSearch", "imageGeneration":
			id, _ := item["id"].(string)
			name := typeName
			var images []string
			if typeName == "Extension" || typeName == "extension" {
				name, _ = item["kind"].(string)
				if name == "" {
					name = typeName
				}
			}
			if typeName == "mcpToolCall" {
				tool, _ := item["tool"].(string)
				if tool != "" {
					name = tool
					if server, _ := item["server"].(string); server != "" {
						name = server + "/" + tool
					}
				}
			}
			if typeName == "dynamicToolCall" {
				tool, _ := item["tool"].(string)
				if namespace, _ := item["namespace"].(string); namespace != "" {
					tool = namespace + "." + tool
				}
				if tool != "" {
					name = tool
				}
			}

			content0 := ""
			// special show diff for filechange
			if typeName == "fileChange" {
				var diffs, paths []string
				if changes, ok := item["changes"].([]any); ok {
					for _, value := range changes {
						change, ok := value.(map[string]any)
						if !ok {
							continue
						}
						if diff, _ := change["diff"].(string); diff != "" {
							diffs = append(diffs, diff)
						}
						path, _ := change["path"].(string)
						kind, _ := change["kind"].(string)
						if value, ok := change["kind"].(map[string]any); ok {
							kind, _ = value["type"].(string)
							if movePath, _ := value["move_path"].(string); movePath != "" {
								kind += " -> " + movePath
							}
						}
						if path != "" || kind != "" {
							paths = append(paths, strings.TrimSpace(path+" "+kind))
						}
					}
				}
				content = strings.Join(diffs, "\n")
				content0 = strings.Join(paths, "\n")
			} else {
				omitResult := typeName == "imageGeneration" ||
					(typeName == "Extension" || typeName == "extension") && name != "image_gen.generation"
				content = _itemJSON(item, omitResult)
			}
			status, _ := item["status"].(string)
			failed := status == "failed" || status == "declined" || item["success"] == false || item["error"] != nil

			// special add image for image gen
			if name == "image_gen.generation" || typeName == "imageGeneration" {
				if result, ok := item["result"].(string); ok && result != "" {
					images = []string{harness.DataURI("image/png", result)}
				}
			}
			return _toolEvent(name, id, content, content0, timestamp, failed, images, true)
		case "subAgentActivity":
			id, _ := item["id"].(string)
			var parts []string
			for _, key := range []string{"kind", "agentPath", "agentThreadId"} {
				if value, _ := item[key].(string); value != "" {
					parts = append(parts, value)
				}
			}
			// Emit complete immediately because started and completed have distinct item IDs.
			return _toolEvent("subagent", id, _itemJSON(item, false), strings.Join(parts, " "), timestamp, false, nil, true)
		case "collabAgentToolCall":
			id, _ := item["id"].(string)
			tool, _ := item["tool"].(string)
			status, _ := item["status"].(string)
			name := "subagent"
			if tool != "" {
				name += ":" + tool
			}
			prompt, ids := _childPrompt(item)
			summary := prompt
			if summary == "" {
				summary = strings.Join(ids, ", ")
			}
			if chars := []rune(summary); len(chars) > 160 {
				summary = string(chars[:160])
			}
			return _toolEvent(name, id, _itemJSON(item, false), summary, timestamp, status == "failed", nil, status != "inProgress")
		case "error":
			return []harness.Event{{Type: harness.EventError, Content: content, TimestampMs: timestamp}}
		}
		return nil
	}

	_rememberChildPrompt := func(item map[string]any) {
		prompt, childIDs := _childPrompt(item)
		if prompt == "" {
			return
		}
		cacheMu.Lock()
		if harnessIDToPrompt[s.RealPath] == nil {
			harnessIDToPrompt[s.RealPath] = map[string]string{}
		}
		for _, childID := range childIDs {
			harnessIDToPrompt[s.RealPath][childID] = prompt
		}
		cacheMu.Unlock()
	}

	// updates Codex partial events with new info
	_upsertItem := func(item map[string]any, timestamp int64, turnID string, completed bool) {
		id, _ := item["id"].(string)
		if id == "" {
			return
		}
		_rememberChildPrompt(item)
		next := _convert(item, timestamp, turnID)
		ref, exists := c.eventRefs[id]
		if exists && ref.completed && !completed {
			return
		}
		if exists {
			timestamp = ref.startedAt
		}
		if timestamp == 0 {
			timestamp = time.Now().UnixMilli()
		}
		s.Mu.Lock()
		if !exists || len(ref.indices) == 0 {
			for _, event := range next {
				ref.indices = append(ref.indices, len(s.Events))
				s.Events = append(s.Events, event)
			}
		} else {
			for i, index := range ref.indices {
				if i < len(next) {
					if !completed && next[i].Type == harness.EventAssistant && s.Events[index].Type == harness.EventAssistant {
						next[i].Content = s.Events[index].Content
					}
					s.Events[index] = next[i]
				}
			}
			if len(next) > len(ref.indices) {
				oldCount := len(ref.indices)
				insertAt := ref.indices[oldCount-1] + 1
				extra := next[oldCount:]
				s.Events = append(s.Events, make([]harness.Event, len(extra))...)
				copy(s.Events[insertAt+len(extra):], s.Events[insertAt:len(s.Events)-len(extra)])
				copy(s.Events[insertAt:], extra)
				for otherID, other := range c.eventRefs {
					for i, index := range other.indices {
						if index >= insertAt {
							other.indices[i] = index + len(extra)
						}
					}
					c.eventRefs[otherID] = other
				}
				for i := oldCount; i < len(next); i++ {
					ref.indices = append(ref.indices, insertAt+i-oldCount)
				}
			}
		}
		ref.startedAt = timestamp
		ref.completed = ref.completed || completed
		c.eventRefs[id] = ref
		s.Mu.Unlock()
		if item["type"] == "userMessage" {
			s.Mu.Lock()
			if s.FirstUserMsg == "" {
				s.FirstUserMsg = _itemText(item)
			}
			s.Mu.Unlock()
		}
	}

	eventBusy, eventBusyKnown := false, false
	_onEvent := func(method string, raw json.RawMessage, includeItemEvents, afterSnapshot bool) {
		var params struct {
			ThreadID    string         `json:"threadId"`
			Item        map[string]any `json:"item"`
			StartedAtMs int64          `json:"startedAtMs"`
			TurnID      string         `json:"turnId"`
			Turn        struct {
				ID        string           `json:"id"`
				StartedAt int64            `json:"startedAt"`
				Items     []map[string]any `json:"items"`
			} `json:"turn"`
			TokenUsage tokenUsage `json:"tokenUsage"`
			Error      struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(raw, &params) != nil || params.ThreadID != c.harnessID {
			return
		}
		switch method {
		case "thread/tokenUsage/updated":
			c.mu.Lock()
			c.usage = params.TokenUsage
			c.mu.Unlock()
		case "turn/started":
			c.mu.Lock()
			c.codexTurnID = params.Turn.ID
			c.mu.Unlock()
			if afterSnapshot {
				eventBusy, eventBusyKnown = true, true
			}
		case "turn/completed":
			if includeItemEvents {
				for _, item := range params.Turn.Items {
					_rememberChildPrompt(item)
					_upsertItem(item, params.Turn.StartedAt*1000, params.Turn.ID, true)
				}
				if afterSnapshot {
					eventBusy, eventBusyKnown = false, true
				}
			}
			c.mu.Lock()
			c.codexTurnID = ""
			c.mu.Unlock()
		case "item/started", "item/completed":
			if includeItemEvents && (method == "item/completed" || params.Item["type"] != "agentMessage") {
				_upsertItem(params.Item, params.StartedAtMs, params.TurnID, method == "item/completed")
			}
		case "item/agentMessage/delta":
			// Keep partial assistant text out of Session.Events; item/completed carries the full message.
		case "error":
			c.mu.Lock()
			c.errors = append(c.errors, harness.Event{Type: harness.EventError, Content: params.Error.Message, TimestampMs: time.Now().UnixMilli()})
			c.mu.Unlock()
		}
	}

	// actual flow of logic
	s.Mu.Lock()
	full := s.DoFullHistory
	s.Mu.Unlock()
	c.mu.Lock()
	full = full || !c.isLoaded
	threadID := c.harnessID
	c.mu.Unlock()
	_processEvents := func(afterSequence uint64, includeBeforeSnapshotItems bool) {
		for _, event := range updates {
			afterSnapshot := event.sequence > afterSequence
			_onEvent(event.method, event.params, includeBeforeSnapshotItems || afterSnapshot, afterSnapshot)
		}
	}

	params := map[string]any{"threadId": threadID}
	raw, responseSequence, err := connRPC0.call("thread/read", params)
	if err != nil {
		// the first few reads when the session is init-ing might fail...
		// try for 30s, else turn the state to busy=false
		if strings.Contains(err.Error(), "failed to read session metadata") {
			c.mu.Lock()
			if c.firstMetadataErrAt.IsZero() {
				c.firstMetadataErrAt = time.Now()
			}
			expired := time.Since(c.firstMetadataErrAt) >= 30*time.Second
			if expired {
				c.firstMetadataErrAt = time.Time{}
				c.codexTurnID = ""
			}

			c.mu.Unlock()
			if !expired {
				return nil
			}
			s.Mu.Lock()
			s.BusyCli = false
			s.Mu.Unlock()
		}
		return err
	}

	c.mu.Lock()
	c.firstMetadataErrAt = time.Time{}
	c.mu.Unlock()
	var result struct {
		Thread struct {
			UpdatedAt      int64  `json:"updatedAt"`
			ParentThreadID string `json:"parentThreadId"`
			Status         struct {
				Type string `json:"type"`
			} `json:"status"`
		} `json:"thread"`
	}
	if err := decode(raw, &result); err != nil {
		return err
	}

	// child prompt lookup
	childPrompt := ""
	if result.Thread.ParentThreadID != "" {
		cacheMu.Lock()
		childPrompt = harnessIDToPrompt[s.RealPath][threadID]
		cacheMu.Unlock()
		if childPrompt == "" {
			pageRaw, _, err := connRPC0.call("thread/turns/list", map[string]any{
				"threadId": result.Thread.ParentThreadID, "limit": 100,
				"sortDirection": "desc", "itemsView": "full",
			})
			if err == nil {
				var page struct {
					Data []struct {
						Items []map[string]any `json:"items"`
					} `json:"data"`
				}
				if decode(pageRaw, &page) == nil {
					for _, turn := range page.Data {
						for _, item := range turn.Items {
							prompt, childIDs := _childPrompt(item)
							for _, childID := range childIDs {
								if childID == threadID && prompt != "" {
									childPrompt = prompt
									cacheMu.Lock()
									if harnessIDToPrompt[s.RealPath] == nil {
										harnessIDToPrompt[s.RealPath] = map[string]string{}
									}
									harnessIDToPrompt[s.RealPath][threadID] = prompt
									cacheMu.Unlock()
									break
								}
							}
							if childPrompt != "" {
								break
							}
						}
						if childPrompt != "" {
							break
						}
					}
				}
			}
		}
		if childPrompt != "" {
			s.Mu.Lock()
			if s.FirstUserMsg == "" {
				s.FirstUserMsg = childPrompt
			}
			s.Mu.Unlock()
		}
	}

	if full {
		var turns []struct {
			ID        string           `json:"id"`
			StartedAt int64            `json:"startedAt"`
			Items     []map[string]any `json:"items"`
		}
		cursor := ""
		for {
			params := map[string]any{"threadId": threadID, "limit": 100, "sortDirection": "asc", "itemsView": "full"}
			if cursor != "" {
				params["cursor"] = cursor
			}
			pageRaw, _, err := connRPC0.call("thread/turns/list", params)
			if err != nil {
				// here we allow some errs to be skipped
				if strings.Contains(err.Error(), "list_turns is not supported yet") || cursor == "" && strings.Contains(err.Error(), "thread/turns/list is unavailable before first user message") {
					break
				}
				return err
			}
			var page struct {
				Data []struct {
					ID        string           `json:"id"`
					StartedAt int64            `json:"startedAt"`
					Items     []map[string]any `json:"items"`
				} `json:"data"`
				NextCursor *string `json:"nextCursor"`
			}
			if err := decode(pageRaw, &page); err != nil {
				return err
			}
			turns = append(turns, page.Data...)
			if page.NextCursor == nil || *page.NextCursor == "" {
				break
			}
			cursor = *page.NextCursor
		}
		eventBusyKnown = false

		events := make([]harness.Event, 0)
		refs := make(map[string]itemEventRef)
		firstUser := ""
		lastTurn := len(turns) - 1
		for ti, turn := range turns {
			for ii, item := range turn.Items {
				_rememberChildPrompt(item)
				status, _ := item["status"].(string)
				active := status == "inProgress" || result.Thread.Status.Type == "active" && ti == lastTurn && ii == len(turn.Items)-1
				if active && item["type"] == "agentMessage" {
					continue
				}
				id, _ := item["id"].(string)
				timestamp := turn.StartedAt * 1000
				converted := _convert(item, timestamp, turn.ID)
				indices := make([]int, 0, len(converted))
				for _, event := range converted {
					indices = append(indices, len(events))
					events = append(events, event)
					if firstUser == "" && event.Type == harness.EventUser {
						firstUser = event.Content
					}
				}
				if id != "" {
					refs[id] = itemEventRef{indices: indices, startedAt: timestamp, completed: !active}
				}
			}
		}

		// Codex child turns can omit a userMessage; use the matched parent prompt instead.
		if firstUser == "" && childPrompt != "" {
			firstUser = childPrompt
			timestamp := int64(0)
			if len(turns) > 0 {
				timestamp = turns[0].StartedAt * 1000
			}
			hasUserEvent := false
			for _, event := range events {
				if event.Type == harness.EventUser {
					hasUserEvent = true
					break
				}
			}
			if !hasUserEvent {
				events = append([]harness.Event{{
					Type: harness.EventUser, Content: childPrompt, TimestampMs: timestamp,
				}}, events...)
				for id, ref := range refs {
					for i := range ref.indices {
						ref.indices[i]++
					}
					refs[id] = ref
				}
			}
		}
		c.eventRefs = refs
		c.mu.Lock()
		c.isLoaded = true
		c.mu.Unlock()
		s.Mu.Lock()
		s.Events = events
		if firstUser != "" {
			s.FirstUserMsg = firstUser
		}
		s.Mu.Unlock()
		_processEvents(responseSequence, true)
	}
	_processEvents(responseSequence, !full)

	c.mu.Lock()
	pendingErrors := c.errors
	c.errors = nil
	usage := c.usage
	activeTurn := c.codexTurnID != ""
	c.mu.Unlock()
	busy := result.Thread.Status.Type == "active" || activeTurn
	if eventBusyKnown {
		busy = eventBusy
	}
	if !busy {
		c.mu.Lock()
		c.codexTurnID = ""
		c.mu.Unlock()
	}

	var fullCost float64
	hasCost := false
	if raw, _, usageErr := connRPC0.call("account/usage/read", map[string]any{"threadId": threadID}); usageErr == nil {
		var result struct {
			ThreadUsage *struct {
				EstimatedUsdMicros      *int64 `json:"estimatedUsdMicros"`
				EstimatedUsageUsdMicros *int64 `json:"estimatedUsageUsdMicros"`
			} `json:"threadUsage"`
		}
		if json.Unmarshal(raw, &result) == nil && result.ThreadUsage != nil {
			usdMicros := result.ThreadUsage.EstimatedUsdMicros
			if usdMicros == nil {
				usdMicros = result.ThreadUsage.EstimatedUsageUsdMicros
			}
			if usdMicros != nil {
				fullCost, hasCost = float64(*usdMicros)/1_000_000, true
			}
		}
	}
	// update all the session state
	s.Mu.Lock()
	s.Events = append(s.Events, pendingErrors...)
	s.BusyCli = busy
	s.DoFullHistory = false
	s.LastTimestampMs = result.Thread.UpdatedAt * 1000
	s.FullInputToken, s.FullOutputToken = usage.Total.InputTokens, usage.Total.OutputTokens
	s.CacheReadToken, s.CacheWriteToken = usage.Total.CachedInputTokens, usage.Total.CacheWriteInputTokens
	s.ContextTokens, s.ContextWindow = usage.Last.TotalTokens, usage.ModelContextWindow
	if hasCost {
		s.FullCost = fullCost
	}
	s.Mu.Unlock()
	committed = true
	return nil
}

// === util fns

func (a *Adapter) sessionConn(s *harness.Session) (*SessionConn, error) {
	cacheMu.Lock()
	s.Mu.Lock()
	id := s.ID
	s.Mu.Unlock()
	c := conns[s.RealPath][id]
	cacheMu.Unlock()
	if c == nil {
		return nil, errors.New("Codex session is not connected")
	}
	if connRPC0 == nil {
		return nil, errors.New("Codex app-server is not connected")
	}
	if err := connRPC0.chkConnection(); err != nil {
		return nil, err
	}
	return c, nil
}

func decode(raw json.RawMessage, out any) error {
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("invalid Codex response: %w", err)
	}
	return nil
}
