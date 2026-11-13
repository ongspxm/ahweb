package opencode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type OpencodeID = string

var (
	eventListenerMu      sync.Mutex
	eventListenerStarted bool
	activityMu           sync.Mutex
	sseCacheMu           sync.Mutex
	sseCacheLocks        sync.Map
	SSECache             = make(map[OpencodeID][]json.RawMessage)
)

func lockSSECache(id OpencodeID) *sync.Mutex {
	value, _ := sseCacheLocks.LoadOrStore(id, &sync.Mutex{})
	mu := value.(*sync.Mutex)
	mu.Lock()
	return mu
}

// === structs
type apiTokens struct {
	Input     int64 `json:"input"`
	Output    int64 `json:"output"`
	Reasoning int64 `json:"reasoning"`
	Cache     struct {
		Read  int64 `json:"read"`
		Write int64 `json:"write"`
	} `json:"cache"`
}

type apiUsage struct {
	Cost   float64   `json:"cost"`
	Tokens apiTokens `json:"tokens"`
}

type apiMessage struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
	Time struct {
		Created   int64 `json:"created"`
		Completed int64 `json:"completed"`
	} `json:"time"`
	Model struct {
		ID         string `json:"id"`
		ProviderID string `json:"providerID"`
	} `json:"model"`
	Files []struct {
		Data   string `json:"data"`
		Mime   string `json:"mime"`
		URL    string `json:"url"`
		URI    string `json:"uri"`
		Source struct {
			Type string `json:"type"`
			URI  string `json:"uri"`
		} `json:"source"`
	} `json:"files"`
	Content []struct {
		Type   string `json:"type"`
		Text   string `json:"text"`
		ID     string `json:"id"`
		CallID string `json:"callID"`
		Name   string `json:"name"`
		Tool   string `json:"tool"`
		URL    string `json:"url"`
		URI    string `json:"uri"`
		MIME   string `json:"mime"`
		State  struct {
			Status  string          `json:"status"`
			Input   json.RawMessage `json:"input"`
			Output  string          `json:"output"`
			Error   json.RawMessage `json:"error"`
			Content []struct {
				Type string `json:"type"`
				URI  string `json:"uri"`
				MIME string `json:"mime"`
			} `json:"content"`
		} `json:"state"`
	} `json:"content"`
	Cost   float64   `json:"cost"`
	Tokens apiTokens `json:"tokens"`
}

// === main entrypoint fns
func (m apiMessage) events() []harness.Event {
	ts := m.Time.Completed
	if ts == 0 {
		ts = m.Time.Created
	}
	model := m.Model.ProviderID + "/" + m.Model.ID
	if m.Type == "user" {
		content, images := m.Text, make([]string, 0)
		for _, file := range m.Files {
			if !strings.HasPrefix(file.Mime, "image/") {
				continue
			}
			uri := file.URL
			if uri == "" {
				uri = file.URI
			}
			if uri == "" {
				uri = file.Source.URI
			}
			if file.Data != "" {
				images = append(images, harness.DataURI(file.Mime, file.Data))
			} else if strings.HasPrefix(uri, "data:image/") {
				images = append(images, uri)
			}
		}
		for _, part := range m.Content {
			if part.Type == "text" && content == "" {
				content = part.Text
			}
			if strings.HasPrefix(part.MIME, "image/") {
				uri := part.URL
				if uri == "" {
					uri = part.URI
				}
				if strings.HasPrefix(uri, "data:image/") {
					images = append(images, uri)
				}
			}
		}
		if content == "" && len(images) == 0 {
			return nil
		}
		return []harness.Event{{Type: harness.EventUser, Content: content, Images: images, TimestampMs: ts, RewindID: m.ID}}
	}
	if m.Type != "assistant" {
		return nil
	}
	out := make([]harness.Event, 0)
	for _, part := range m.Content {
		if (part.Type == "image" || part.Type == "file") && strings.HasPrefix(part.MIME, "image/") {
			uri := part.URL
			if uri == "" {
				uri = part.URI
			}
			if strings.HasPrefix(uri, "data:image/") {
				out = append(out, harness.Event{Type: harness.EventAssistant, Images: []string{uri}, TimestampMs: ts, Model: model})
			} else if uri != "" {
				out = append(out, harness.Event{Type: harness.EventAssistant, Images: []string{harness.DataURI(part.MIME, uri)}, TimestampMs: ts, Model: model})
			}
		}
		switch part.Type {
		case "reasoning":
			if part.Text != "" {
				out = append(out, harness.Event{Type: harness.EventThinking, Content: part.Text, TimestampMs: ts, Model: model})
			}
		case "text":
			if part.Text != "" {
				out = append(out, harness.Event{Type: harness.EventAssistant, Content: part.Text, TimestampMs: ts, Model: model})
			}
		case "tool":
			id := part.CallID
			if id == "" {
				id = part.ID
			}
			name := part.Name
			if name == "" {
				name = part.Tool
			}
			input := string(part.State.Input)
			if input == "" || input == "null" {
				input = "{}"
			}
			switch part.State.Status {
			case "pending", "running", "streaming", "completed", "error":
				out = append(out, harness.Event{Type: harness.EventToolStart, ToolName: name, ToolCallID: id, Content: input, TimestampMs: ts})
				if part.State.Status == "completed" || part.State.Status == "error" {
					content, failed := part.State.Output, part.State.Status == "error"
					if failed {
						content = string(part.State.Error)
					}
					var images []string
					for _, item := range part.State.Content {
						if strings.HasPrefix(item.MIME, "image/") && strings.HasPrefix(item.URI, "data:image/") {
							images = append(images, item.URI)
						}
					}
					out = append(out, harness.Event{Type: harness.EventToolEnd, ToolName: name, ToolCallID: id, Content: content, Images: images, ToolFailed: failed, TimestampMs: ts})
				}
			}
		}
	}
	return out
}

func processSSECache(id OpencodeID, discardIncomplete bool) ([]apiMessage, []apiUsage, bool) {
	// called on each sessions's Poll(), will merge raw msgs and keep incomplete msgs cached.
	cacheMu := lockSSECache(id)
	defer cacheMu.Unlock()

	// all the setup stuff
	sseCacheMu.Lock()
	raws := append([]json.RawMessage(nil), SSECache[id]...)
	sseCacheMu.Unlock()

	type ssePartial struct {
		info  json.RawMessage
		parts map[string]json.RawMessage
		order []string
		raws  []json.RawMessage
	}
	partials := make(map[string]*ssePartial)
	order := make([]string, 0)
	_ensure := func(mid string) *ssePartial {
		if partials[mid] == nil {
			order = append(order, mid)
		}
		if partials[mid] == nil {
			partials[mid] = &ssePartial{parts: make(map[string]json.RawMessage)}
		}
		return partials[mid]
	}
	_putSSEPart := func(partial *ssePartial, id string, encoded json.RawMessage) {
		if _, exists := partial.parts[id]; !exists {
			partial.order = append(partial.order, id)
		}
		partial.parts[id] = encoded
	}

	// === main logic for recrusive parsing
	var _walk func(any, map[string]bool)
	_walk = func(value any, ids map[string]bool) {
		switch item := value.(type) {
		case map[string]any:
			if info, ok := item["info"].(map[string]any); ok {
				if mid, _ := info["id"].(string); mid != "" {
					partial := _ensure(mid)
					partial.info, _ = json.Marshal(info)
					ids[mid] = true
					if parts, ok := item["parts"].([]any); ok {
						for index, part := range parts {
							key := fmt.Sprint(index)
							if object, ok := part.(map[string]any); ok {
								if id, ok := object["id"].(string); ok && id != "" {
									key = id
								}
							}
							encoded, err := json.Marshal(part)
							if err == nil {
								_putSSEPart(partial, key, encoded)
							}
						}
					}
				}
			}
			if part, ok := item["part"].(map[string]any); ok {
				if mid, _ := part["messageID"].(string); mid != "" {
					pid, _ := part["id"].(string)
					if pid == "" {
						pid = fmt.Sprint(len(_ensure(mid).parts))
					}
					encoded, err := json.Marshal(part)
					if err == nil {
						_putSSEPart(_ensure(mid), pid, encoded)
					}
					ids[mid] = true
				}
			}
			if mid, _ := item["messageID"].(string); mid != "" {
				if delta, ok := item["delta"].(string); ok {
					pid, _ := item["partID"].(string)
					if pid == "" {
						pid = "delta"
					}
					partial := _ensure(mid)
					var part map[string]any
					_ = json.Unmarshal(partial.parts[pid], &part)
					if part == nil {
						part = make(map[string]any)
						partial.order = append(partial.order, pid)
					}
					part["type"] = "text"
					part["id"] = pid
					previous, _ := part["text"].(string)
					part["text"] = previous + delta
					if encoded, err := json.Marshal(part); err == nil {
						partial.parts[pid] = encoded
					}
					ids[mid] = true
				}
			}
			for _, child := range item {
				_walk(child, ids)
			}
		case []any:
			for _, child := range item {
				_walk(child, ids)
			}
		}
	}

	// pull out usage stats
	var usages []apiUsage
	for _, raw := range raws {
		var event struct {
			Type       string   `json:"type"`
			Properties apiUsage `json:"properties"`
		}
		if json.Unmarshal(raw, &event) == nil && event.Type == "session.step.ended" {
			usages = append(usages, event.Properties)
		}
		var value any
		if json.Unmarshal(raw, &value) != nil {
			continue
		}
		ids := make(map[string]bool)
		_walk(value, ids)
		for mid := range ids {
			partials[mid].raws = append(partials[mid].raws, append(json.RawMessage(nil), raw...))
		}
	}

	// forming the final message output
	messages, retained := make([]apiMessage, 0), make([]json.RawMessage, 0)
	for _, mid := range order {
		partial := partials[mid]
		var info struct {
			Role string `json:"role"`
			Time struct {
				Created   int64 `json:"created"`
				Completed int64 `json:"completed"`
			} `json:"time"`
			ModelID    string          `json:"modelID"`
			ProviderID string          `json:"providerID"`
			Cost       float64         `json:"cost"`
			Tokens     apiTokens       `json:"tokens"`
			Files      json.RawMessage `json:"files"`
		}
		_ = json.Unmarshal(partial.info, &info)
		if len(partial.info) == 0 || len(partial.parts) == 0 || (info.Role != "user" && info.Time.Completed == 0) {
			if !discardIncomplete {
				retained = append(retained, partial.raws...)
			}
			continue
		}
		parts := make([]json.RawMessage, 0, len(partial.parts))
		for _, key := range partial.order {
			if part := partial.parts[key]; len(part) > 0 {
				parts = append(parts, part)
			}
		}
		message := map[string]any{
			"id":     mid,
			"type":   info.Role,
			"time":   info.Time,
			"model":  map[string]string{"id": info.ModelID, "providerID": info.ProviderID},
			"cost":   info.Cost,
			"tokens": info.Tokens,
		}
		if info.Role == "user" {
			var text strings.Builder
			for _, raw := range parts {
				var part struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}
				if json.Unmarshal(raw, &part) == nil && part.Type == "text" {
					text.WriteString(part.Text)
				}
			}
			message["text"] = text.String()
			message["content"] = parts
			if len(info.Files) > 0 {
				message["files"] = info.Files
			}
		} else {
			message["content"] = parts
		}
		encoded, _ := json.Marshal(message)
		var parsed apiMessage
		if json.Unmarshal(encoded, &parsed) == nil {
			messages = append(messages, parsed)
		}
	}
	sseCacheMu.Lock()
	SSECache[id] = retained
	sseCacheMu.Unlock()
	return messages, usages, len(retained) > 0
}

func ensureEventListenerUp() {
	serviceMu.Lock()
	base, pass := serviceURL, password
	serviceMu.Unlock()
	if base == "" {
		return
	}
	eventListenerMu.Lock()
	if eventListenerStarted {
		eventListenerMu.Unlock()
		return
	}
	eventListenerStarted = true
	eventListenerMu.Unlock()

	// do busy state snapshot every 30s
	// in case we missed something or in a bad state
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			activityMu.Lock()
			var active struct {
				Data map[string]struct {
					Type string `json:"type"`
				} `json:"data"`
			}
			if err := (&Conn{}).call(http.MethodGet, "/api/session/active", nil, nil, &active); err == nil {
				activeIDs := make(map[string]bool, len(active.Data))
				for id := range active.Data {
					activeIDs[id] = true
				}
				cacheMu.Lock()
				var sessions []struct {
					id      string
					session *harness.Session
				}
				for _, byID := range cache {
					for _, conn := range byID {
						if conn.session != nil {
							sessions = append(sessions, struct {
								id      string
								session *harness.Session
							}{conn.OpencodeID, conn.session})
						}
					}
				}
				cacheMu.Unlock()
				for _, item := range sessions {
					item.session.Mu.Lock()
					wasBusy := item.session.BusyCli
					busy := activeIDs[item.id]
					item.session.BusyCli = busy
					if wasBusy && !busy {
						item.session.DoFullHistory = true
					}
					item.session.Mu.Unlock()
				}
			}
			activityMu.Unlock()
		}
	}()

	// sse loop
	go func() {
		connected := false
		for {
			req, err := http.NewRequest(http.MethodGet, base+"/api/event", nil)
			if err == nil {
				req.SetBasicAuth("opencode", pass)
				resp, requestErr := (&http.Client{}).Do(req)
				if requestErr == nil {
					if resp.StatusCode != http.StatusOK || !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
						_ = resp.Body.Close()
						time.Sleep(time.Second)
						continue
					}
					connected = true
					data := ""
					scanner := bufio.NewScanner(resp.Body)
					scanner.Buffer(make([]byte, 4096), 1024*1024)
					for scanner.Scan() {
						line := scanner.Text()
						if line == "" {
							if data == "" {
								continue
							}
							raw := json.RawMessage(strings.TrimSuffix(data, "\n"))
							var event map[string]any
							if json.Unmarshal(raw, &event) == nil {
								var _findSession func(any) string
								_findSession = func(value any) string {
									switch item := value.(type) {
									case map[string]any:
										if id, ok := item["sessionID"].(string); ok {
											return id
										}
										for _, child := range item {
											if id := _findSession(child); id != "" {
												return id
											}
										}
									case []any:
										for _, child := range item {
											if id := _findSession(child); id != "" {
												return id
											}
										}
									}
									return ""
								}
								id := _findSession(event)
								payload := event
								if nested, ok := event["payload"].(map[string]any); ok {
									payload = nested
								}
								busy, ok := false, false
								if id != "" {
									typeName, _ := payload["type"].(string)
									switch {
									case typeName == "session.status":
										properties, _ := payload["properties"].(map[string]any)
										if properties == nil {
											properties, _ = payload["data"].(map[string]any)
										}
										status, _ := properties["status"].(map[string]any)
										state, _ := status["type"].(string)
										busy, ok = state != "idle", state != ""
									case typeName == "session.execution.started":
										busy, ok = true, true
									case strings.HasPrefix(typeName, "session.execution."):
										ok = true
									default:
										busy, ok = true, true
									}
								}
								if ok {
									sessionCacheMu := lockSSECache(id)
									sseCacheMu.Lock()
									SSECache[id] = append(SSECache[id], append(json.RawMessage(nil), raw...))
									sseCacheMu.Unlock()
									sessionCacheMu.Unlock()
									activityMu.Lock()
									cacheMu.Lock()
									var sessions []*harness.Session
									for _, byID := range cache {
										for _, conn := range byID {
											if conn.OpencodeID == id && conn.session != nil {
												sessions = append(sessions, conn.session)
											}
										}
									}
									cacheMu.Unlock()
									for _, session := range sessions {
										session.Mu.Lock()
										session.BusyCli = busy
										if !busy {
											session.DoFullHistory = true
										}
										session.Mu.Unlock()
									}
									activityMu.Unlock()
								}
							}
							data = ""
						}
						if value, ok := strings.CutPrefix(line, "data:"); ok {
							data += strings.TrimSpace(value) + "\n"
						}
					}
					_ = resp.Body.Close()
				}
			}
			if connected {
				var sessions []*harness.Session
				cacheMu.Lock()
				for _, byID := range cache {
					for _, conn := range byID {
						if conn.session != nil {
							sessions = append(sessions, conn.session)
						}
					}
				}
				cacheMu.Unlock()
				for _, session := range sessions {
					session.Mu.Lock()
					session.DoFullHistory = true
					session.Mu.Unlock()
				}
			}
			time.Sleep(time.Second)
		}
	}()
}
