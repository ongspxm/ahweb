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

type Adapter struct{ harness.DefaultAdapter }

type itemEventRef struct {
	indices   []int
	startedAt int64
	completed bool
}

type conn struct {
	mu                 sync.Mutex
	codexThreadID      string
	codexTurnID        string
	isWriter           bool
	firstMetadataErrAt time.Time
	usage              tokenUsage
	errors             []harness.Event
	isLoaded           bool
	eventRefs          map[string]itemEventRef
	eventCache         []cachedEvent
}

var (
	cacheMu   sync.Mutex
	conns     = map[string]map[string]*conn{}
	threadIDs = map[string]map[string]string{}
)

func getConn(s *harness.Session) (*conn, error) {
	cacheMu.Lock()
	s.Mu.Lock()
	id := s.ID
	s.Mu.Unlock()
	c := conns[s.RealPath][id]
	cacheMu.Unlock()
	if c == nil {
		return nil, errors.New("Codex session is not connected")
	}
	if processRPC == nil {
		return nil, errors.New("Codex App Server is not connected")
	}
	if err := processRPC.status(); err != nil {
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

func (*Adapter) Ready() error { _, err := openStream(); return err }

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
		threadID = threadIDs[s.RealPath][id]
	}
	cacheMu.Unlock()
	// History refreshes must not resume a healthy, possibly unsaved thread.
	if cached != nil {
		if processRPC != nil && processRPC.status() == nil {
			return nil
		}
	}

	// startup the connection
	_ = a.CleanUp(s)
	c, err := openStream()
	if err != nil {
		return err
	}
	method := "thread/start"
	params := map[string]any{"cwd": s.RealPath}
	if threadID != "" {
		method = "thread/read"
		params = map[string]any{"threadId": threadID}
	}
	raw, _, err := processRPC.call(method, params)
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
	c.codexThreadID = result.Thread.ID
	c.isWriter = method == "thread/start"
	c.mu.Unlock()
	processRPC.mu.Lock()
	processRPC.threads[result.Thread.ID] = c
	processRPC.mu.Unlock()
	cacheMu.Lock()

	if isLocal {
		if threadIDs[s.RealPath] == nil {
			threadIDs[s.RealPath] = map[string]string{}
		}
		threadIDs[s.RealPath][id] = result.Thread.ID
	}
	if conns[s.RealPath] == nil {
		conns[s.RealPath] = map[string]*conn{}
	}
	conns[s.RealPath][id] = c
	cacheMu.Unlock()
	return nil
}

func (a *Adapter) SendPrompt(s *harness.Session) error {
	c, err := getConn(s)
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
	threadID, isWriter := c.codexThreadID, c.isWriter
	c.mu.Unlock()

	// swap the connection to write (if needed)
	if !isWriter {
		if _, _, err := processRPC.call("thread/resume", map[string]any{"threadId": threadID, "cwd": s.RealPath}); err != nil {
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
	raw, _, err := processRPC.call("turn/start", params)
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
	c, err := getConn(s)
	if err != nil {
		return nil
	}
	c.mu.Lock()
	turn := c.codexTurnID
	c.mu.Unlock()
	if turn == "" {
		return nil
	}
	_, _, err = processRPC.call("turn/interrupt", map[string]any{"threadId": c.codexThreadID, "turnId": turn})
	return err
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
		threadID, isWriter := c.codexThreadID, c.isWriter
		c.mu.Unlock()

		// if its writer, we actually unsub to try to give back control to other clients
		if processRPC != nil {
			if isWriter && processRPC.status() == nil {
				_, _, err := processRPC.call("thread/unsubscribe", map[string]any{"threadId": threadID})
				if err != nil {
					return err
				}
			}
			processRPC.mu.Lock()
			delete(processRPC.threads, threadID)
			processRPC.mu.Unlock()
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

func (*Adapter) GetModels() (map[string]harness.Model, error) {
	_, err := openStream()
	if err != nil {
		return nil, err
	}
	models := map[string]harness.Model{}
	cursor := ""
	for {
		params := map[string]any{"limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, _, err := processRPC.call("model/list", params)
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
	_, err := openStream()
	if err != nil {
		return nil, err
	}
	var sessions []harness.SessionLite
	cacheMu.Lock()
	mappedThreads := map[string]bool{}
	for _, id := range threadIDs[realFolder] {
		mappedThreads[id] = true
	}
	cacheMu.Unlock()
	cursor := ""
	for {
		params := map[string]any{"cwd": realFolder, "limit": 100, "sortKey": "updated_at", "sortDirection": "desc"}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, _, err := processRPC.call("thread/list", params)
		if err != nil {
			return nil, err
		}
		var result struct {
			Data []struct {
				ID        string `json:"id"`
				Name      string `json:"name"`
				Preview   string `json:"preview"`
				UpdatedAt int64  `json:"updatedAt"`
				ParentID  string `json:"parentThreadId"`
			} `json:"data"`
			NextCursor *string `json:"nextCursor"`
		}
		if err := decode(raw, &result); err != nil {
			return nil, err
		}
		for _, thread := range result.Data {
			if mappedThreads[thread.ID] {
				continue
			}
			lite := harness.SessionLite{ID: thread.ID, RealPath: realFolder, Harness: "codex", Name: thread.Name, FirstUserMsg: thread.Preview, LastTimestampMs: thread.UpdatedAt * 1000, ParentSessID: thread.ParentID}
			sessions = append(sessions, lite)
		}
		if result.NextCursor == nil || *result.NextCursor == "" {
			break
		}
		cursor = *result.NextCursor
	}
	return sessions, nil
}

func (a *Adapter) Poll(s *harness.Session) error {
	c, err := getConn(s)
	if err != nil {
		return err
	}

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

	_convert := func(item map[string]any, timestamp int64) []harness.Event {
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
			return []harness.Event{{Type: harness.EventUser, Content: content, Images: images, TimestampMs: timestamp}}
		case "agentMessage":
			return []harness.Event{{Type: harness.EventAssistant, Content: content, TimestampMs: timestamp}}
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
			start := harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: id, Content: cmd, TimestampMs: timestamp}
			if status == "inProgress" {
				return []harness.Event{start}
			}
			output, _ := item["aggregatedOutput"].(string)
			return []harness.Event{start, {Type: harness.EventToolEnd, ToolName: name, ToolCallID: id, ToolFailed: status == "failed", Content: output, TimestampMs: timestamp}}
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
			if typeName == "Extension" || typeName == "extension" || typeName == "imageGeneration" {
				fields := make(map[string]any, len(item))
				for key, value := range item {
					if key != "type" && key != "id" && key != "kind" && key != "status" && (key != "result" || name != "image_gen.generation" && typeName != "imageGeneration") {
						fields[key] = value
					}
				}
				if raw, err := json.MarshalIndent(fields, "", "  "); err == nil {
					content = string(raw)
				}
			}
			if name == "image_gen.generation" || typeName == "imageGeneration" {
				if result, ok := item["result"].(string); ok && result != "" {
					images = []string{harness.DataURI("image/png", result)}
				}
			}
			return []harness.Event{{Type: harness.EventToolStart, ToolName: name, ToolCallID: id, Content: content, TimestampMs: timestamp}, {Type: harness.EventToolEnd, ToolName: name, ToolCallID: id, Content: content, Images: images, TimestampMs: timestamp}}
		case "error":
			return []harness.Event{{Type: harness.EventError, Content: content, TimestampMs: timestamp}}
		}
		return nil
	}

	// updates Codex partial events with new info
	_upsertItem := func(item map[string]any, timestamp int64, completed bool) {
		id, _ := item["id"].(string)
		if id == "" {
			return
		}
		next := _convert(item, timestamp)
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
		if json.Unmarshal(raw, &params) != nil || params.ThreadID != c.codexThreadID {
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
					_upsertItem(item, params.Turn.StartedAt*1000, true)
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
				_upsertItem(params.Item, params.StartedAtMs, method == "item/completed")
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
	threadID := c.codexThreadID
	c.mu.Unlock()
	_processEvents := func(afterSequence uint64, includeBeforeSnapshotItems bool) {
		c.mu.Lock()
		events := c.eventCache
		c.eventCache = nil
		c.mu.Unlock()
		for _, event := range events {
			afterSnapshot := event.sequence > afterSequence
			_onEvent(event.method, event.params, includeBeforeSnapshotItems || afterSnapshot, afterSnapshot)
		}
	}

	params := map[string]any{"threadId": threadID}
	raw, responseSequence, err := processRPC.call("thread/read", params)
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
			Name      string `json:"name"`
			UpdatedAt int64  `json:"updatedAt"`
			Status    struct {
				Type string `json:"type"`
			} `json:"status"`
		} `json:"thread"`
	}
	if err := decode(raw, &result); err != nil {
		return err
	}

	if full {
		var turns []struct {
			StartedAt int64            `json:"startedAt"`
			Items     []map[string]any `json:"items"`
		}
		cursor := ""
		for {
			params := map[string]any{"threadId": threadID, "limit": 100, "sortDirection": "asc", "itemsView": "full"}
			if cursor != "" {
				params["cursor"] = cursor
			}
			pageRaw, _, err := processRPC.call("thread/turns/list", params)
			if err != nil {
				if strings.Contains(err.Error(), "list_turns is not supported yet") || cursor == "" && strings.Contains(err.Error(), "thread/turns/list is unavailable before first user message") {
					break
				}
				return err
			}
			var page struct {
				Data []struct {
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
				status, _ := item["status"].(string)
				active := status == "inProgress" || result.Thread.Status.Type == "active" && ti == lastTurn && ii == len(turn.Items)-1
				if active && item["type"] == "agentMessage" {
					continue
				}
				id, _ := item["id"].(string)
				timestamp := turn.StartedAt * 1000
				converted := _convert(item, timestamp)
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
		c.eventRefs = refs
		c.mu.Lock()
		c.isLoaded = true
		c.mu.Unlock()
		s.Mu.Lock()
		s.Events = events
		s.FirstUserMsg = firstUser
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
	if raw, _, usageErr := processRPC.call("account/usage/read", map[string]any{"threadId": threadID}); usageErr == nil {
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
	s.Name = result.Thread.Name
	s.LastTimestampMs = result.Thread.UpdatedAt * 1000
	s.FullInputToken, s.FullOutputToken = usage.Total.InputTokens, usage.Total.OutputTokens
	s.CacheReadToken, s.CacheWriteToken = usage.Total.CachedInputTokens, usage.Total.CacheWriteInputTokens
	s.ContextTokens, s.ContextWindow = usage.Last.TotalTokens, usage.ModelContextWindow
	if hasCost {
		s.FullCost = fullCost
	}
	s.Mu.Unlock()
	return nil
}
