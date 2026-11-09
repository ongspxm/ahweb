package opencode

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

var (
	cacheMu       sync.Mutex
	toOpencodeIDs = make(map[string]string)
	toLocalIDs    = make(map[string]string)
	cache         = make(map[string]map[string]*Conn)

	serviceMu  sync.Mutex
	serviceURL string
	password   string

	// session it self doesnt give model context limit, this is a global map
	modelLimits = make(map[string]int64)
)

type Adapter struct{ harness.DefaultAdapter }

func (*Adapter) Ready() error {
	serviceMu.Lock()
	defer serviceMu.Unlock()
	if serviceURL != "" {
		return nil
	}
	if _, err := exec.LookPath("opencode"); err != nil {
		return err
	}
	status, err := exec.Command("opencode", "service", "status").CombinedOutput()
	if err != nil {
		out, startErr := exec.Command("opencode", "service", "start").CombinedOutput()
		if startErr != nil {
			return fmt.Errorf("could not start opencode service: %s: %w", strings.TrimSpace(string(out)), startErr)
		}
		status, err = exec.Command("opencode", "service", "status").CombinedOutput()
		if err != nil {
			return fmt.Errorf("could not discover opencode service: %s: %w", strings.TrimSpace(string(status)), err)
		}
	}
	var base string
	for _, field := range strings.Fields(string(status)) {
		field = strings.Trim(field, "\"'(),")
		if strings.HasPrefix(field, "http://") || strings.HasPrefix(field, "https://") {
			base = strings.TrimRight(field, "/")
			break
		}
	}
	if base == "" {
		return fmt.Errorf("opencode service status did not return a URL: %s", strings.TrimSpace(string(status)))
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("could not find user config directory: %w", err)
	}
	pass := ""
	raw, err := os.ReadFile(filepath.Join(configDir, "opencode", "service.json"))
	if err == nil {
		var config struct {
			Password string `json:"password"`
		}
		if err := json.Unmarshal(raw, &config); err != nil {
			return fmt.Errorf("could not read opencode service config: %w", err)
		}
		pass = config.Password
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("could not read opencode service config: %w", err)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Get(base + "/global/health")
	if err != nil {
		return fmt.Errorf("opencode service is unreachable: %w", err)
	}
	_ = resp.Body.Close()
	serviceURL, password = base, pass
	return nil
}

// === conn stuff
type Conn struct {
	OpencodeID  string
	LastUpdated int64
	Fetched     bool
	session     *harness.Session
}

func (c *Conn) call(method, path string, query url.Values, body any, result any) error {
	if c.OpencodeID != "" {
		path = "/api/session/" + url.PathEscape(c.OpencodeID) + path
	}
	for attempt := 0; attempt < 2; attempt++ {
		if err := (&Adapter{}).Ready(); err != nil {
			return err
		}
		serviceMu.Lock()
		base, pass := serviceURL, password
		serviceMu.Unlock()
		ensureEventListenerUp()
		var reader io.Reader
		if body != nil {
			raw, err := json.Marshal(body)
			if err != nil {
				return err
			}
			reader = bytes.NewReader(raw)
		}
		target := base + path
		if len(query) > 0 {
			target += "?" + query.Encode()
		}
		req, err := http.NewRequest(method, target, reader)
		if err != nil {
			return err
		}
		req.SetBasicAuth("opencode", pass)
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
		if err != nil || resp.StatusCode == http.StatusUnauthorized {
			if resp != nil {
				_ = resp.Body.Close()
			}
			serviceMu.Lock()
			serviceURL, password = "", ""
			serviceMu.Unlock()
			if attempt == 0 {
				continue
			}
			if err != nil {
				return fmt.Errorf("opencode request failed: %w", err)
			}
			return errors.New("opencode service rejected basic authentication")
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("opencode %s %s: %s", method, path, strings.TrimSpace(string(raw)))
		}
		if result != nil && len(raw) > 0 {
			if err := json.Unmarshal(raw, result); err != nil {
				return fmt.Errorf("could not decode opencode response: %w", err)
			}
		}
		return nil
	}
	return errors.New("opencode service unavailable")
}

// === actual adater fns
func (a *Adapter) conn(s *harness.Session) (*Conn, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if conn := cache[s.RealPath][s.ID]; conn != nil {
		return conn, nil
	}
	return nil, errors.New("opencode session is not connected")
}

func (a *Adapter) Connect(s *harness.Session, force bool) error {
	if err := a.Ready(); err != nil {
		return err
	}
	cacheMu.Lock()
	conn := cache[s.RealPath][s.ID]
	if conn != nil && !force {
		conn.session = s
		cacheMu.Unlock()
		ensureEventListenerUp()
		return nil
	}
	cacheMu.Unlock()
	OpencodeID := ""
	if strings.HasPrefix(s.ID, "ses") {
		OpencodeID = s.ID
	} else {
		cacheMu.Lock()
		OpencodeID = toOpencodeIDs[s.ID]
		cacheMu.Unlock()
	}

	// for new sess, create OpencodeID and save mapping
	if OpencodeID == "" {
		body := map[string]any{"location": map[string]string{"directory": s.RealPath}}
		s.Mu.Lock()
		model, reasoning := s.PromptModel, s.PromptReasoning
		s.Mu.Unlock()
		if model != nil {
			provider, id, ok := strings.Cut(model.ID, "/")
			if !ok || provider == "" || id == "" {
				return fmt.Errorf("invalid opencode model ID: %q", model.ID)
			}
			selected := map[string]any{"providerID": provider, "id": id}
			if reasoning != "" {
				selected["variant"] = reasoning
			}
			body["model"] = selected
		}
		var created struct {
			Data struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := (&Conn{}).call(http.MethodPost, "/api/session", nil, body, &created); err != nil {
			return err
		}
		OpencodeID = created.Data.ID
		if OpencodeID == "" {
			return errors.New("opencode returned no session ID")
		}
		cacheMu.Lock()
		toOpencodeIDs[s.ID] = OpencodeID
		toLocalIDs[OpencodeID] = s.ID
		cacheMu.Unlock()
	}

	conn = &Conn{OpencodeID: OpencodeID, session: s}
	cacheMu.Lock()
	if cache[s.RealPath] == nil {
		cache[s.RealPath] = make(map[string]*Conn)
	}
	cache[s.RealPath][s.ID] = conn
	cacheMu.Unlock()
	s.Mu.Lock()
	s.DoFullHistory = true
	s.Mu.Unlock()
	ensureEventListenerUp()
	return nil
}

func (a *Adapter) SendPrompt(s *harness.Session) error {
	conn, err := a.conn(s)
	if err != nil {
		return err
	}
	s.Mu.Lock()
	model, variant, prompt := s.PromptModel, s.PromptReasoning, s.SendingMsg
	images := append([]string(nil), s.SendingImages...)
	s.Mu.Unlock()
	body := map[string]any{"text": prompt}
	if len(images) > 0 {
		files := make([]map[string]string, 0, len(images))
		for _, uri := range images {
			if _, err := harness.ParseImage(uri); err != nil {
				return err
			}
			files = append(files, map[string]string{"uri": uri})
		}
		body["files"] = files
	}
	if model != nil {
		provider, id, ok := strings.Cut(model.ID, "/")
		if !ok || provider == "" || id == "" {
			return fmt.Errorf("invalid opencode model ID: %q", model.ID)
		}
		selected := map[string]any{"id": id, "providerID": provider}
		if variant != "" {
			selected["variant"] = variant
		}
		if err := conn.call(http.MethodPost, "/model", nil, map[string]any{"model": selected}, nil); err != nil {
			return err
		}
	}
	if err := conn.call(http.MethodPost, "/prompt", nil, body, nil); err != nil {
		return err
	}
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
	return conn.call(http.MethodPost, "/interrupt", nil, nil, nil)
}

func (*Adapter) CleanUp(s *harness.Session) error {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	delete(cache[s.RealPath], s.ID)
	if len(cache[s.RealPath]) == 0 {
		delete(cache, s.RealPath)
	}
	return nil
}

func (a *Adapter) GetModels() (map[string]harness.Model, error) {
	var response struct {
		Data []struct {
			ModelID    string `json:"modelID"`
			ProviderID string `json:"providerID"`
			Limit      struct {
				Context int64 `json:"context"`
			} `json:"limit"`
			Variants []struct {
				ID string `json:"id"`
			} `json:"variants"`
		} `json:"data"`
	}
	if err := (&Conn{}).call(http.MethodGet, "/api/model", nil, nil, &response); err != nil {
		return nil, err
	}
	models, limits := make(map[string]harness.Model), make(map[string]int64)
	for _, model := range response.Data {
		id := model.ProviderID + "/" + model.ModelID
		entry := harness.Model{Harness: "opencode", ID: id}
		for _, variant := range model.Variants {
			entry.Reasoning = append(entry.Reasoning, variant.ID)
		}
		models[id], limits[id] = entry, model.Limit.Context
	}
	cacheMu.Lock()
	modelLimits = limits
	cacheMu.Unlock()
	return models, nil
}

func (a *Adapter) Poll(s *harness.Session) error {
	var input, output, read, write int64
	var cost float64
	var lastMessage, lastAssistant apiMessage
	var processedFullMessages, hasAssistant bool

	_updateStats := func(title string, updated int64) {
		s.Mu.Lock()
		defer s.Mu.Unlock()
		if title != "" {
			s.Name = title
		}
		s.LastTimestampMs = updated
		if !processedFullMessages {
			return
		}

		// this one is accumulated, but in partial poll, this will skip
		s.FullInputToken, s.FullOutputToken, s.CacheReadToken, s.CacheWriteToken, s.FullCost = input, output, read, write, cost
		last := lastMessage
		if hasAssistant {
			last = lastAssistant
		}
		tokens := last.Tokens
		s.ContextTokens = tokens.Input + tokens.Output + tokens.Reasoning + tokens.Cache.Read
		cacheMu.Lock()
		s.ContextWindow = modelLimits[last.Model.ProviderID+"/"+last.Model.ID]
		cacheMu.Unlock()
	}

	conn, err := a.conn(s)
	if err != nil {
		return err
	}
	var info struct {
		Data struct {
			Title string `json:"title"`
			Time  struct {
				Updated int64 `json:"updated"`
			} `json:"time"`
		} `json:"data"`
	}
	if err := conn.call(http.MethodGet, "", nil, nil, &info); err != nil {
		return err
	}
	update := info.Data.Time.Updated
	title := info.Data.Title
	if title == conn.OpencodeID {
		title = ""
	}

	s.Mu.Lock()
	full, busy := s.DoFullHistory, s.BusyCli
	s.Name = title
	s.LastTimestampMs = update
	s.Mu.Unlock()

	// partial polling -- processing SSE msgs only
	if !full {
		sseMessages, usages, pending := processSSECache(conn.OpencodeID, false)
		for _, usage := range usages {
			s.Mu.Lock()
			s.FullInputToken += usage.Tokens.Input
			s.FullOutputToken += usage.Tokens.Output
			s.CacheReadToken += usage.Tokens.Cache.Read
			s.CacheWriteToken += usage.Tokens.Cache.Write
			s.FullCost += usage.Cost
			s.ContextTokens = usage.Tokens.Input + usage.Tokens.Output + usage.Tokens.Reasoning + usage.Tokens.Cache.Read
			s.Mu.Unlock()
		}
		partial := make([]harness.Event, 0)
		for _, message := range sseMessages {
			partial = append(partial, message.events()...)
		}
		if len(partial) > 0 {
			s.Mu.Lock()
			s.Events = append(s.Events, partial...)
			if s.FirstUserMsg == "" {
				for _, event := range partial {
					if event.Type == harness.EventUser {
						s.FirstUserMsg = event.Content
						break
					}
				}
			}
			s.Mu.Unlock()
			conn.Fetched, conn.LastUpdated = true, update
			return nil
		}

		if conn.Fetched && conn.LastUpdated == update {
			if pending || busy {
				return nil
			}
			_updateStats(title, update)
			return nil
		}
	}

	// full polling; drop older SSE events first, leaving new arrivals cached.
	// but before that, clear sse cache first
	cursor := ""
	events := make([]harness.Event, 0)
	messages := make([]apiMessage, 0)

	sessionCacheMu := lockSSECache(conn.OpencodeID)
	sseCacheMu.Lock()
	delete(SSECache, conn.OpencodeID)
	sseCacheMu.Unlock()
	sessionCacheMu.Unlock()
	var first string
	for {
		q := url.Values{"limit": {"200"}}
		if cursor == "" {
			q.Set("order", "asc")
		} else {
			q.Set("cursor", cursor)
		}
		var page struct {
			Data   []apiMessage `json:"data"`
			Cursor struct {
				Next *string `json:"next"`
			} `json:"cursor"`
		}
		if err := conn.call(http.MethodGet, "/message", q, nil, &page); err != nil {
			return err
		}
		if len(page.Data) == 0 {
			break
		}
		messages = append(messages, page.Data...)
		if page.Cursor.Next == nil {
			break
		}
		cursor = *page.Cursor.Next
	}
	for _, message := range messages {
		processedFullMessages = true
		usage := message.Tokens
		input += usage.Input
		output += usage.Output
		read += usage.Cache.Read
		write += usage.Cache.Write
		cost += message.Cost
		lastMessage = message
		if message.Type == "assistant" {
			lastAssistant, hasAssistant = message, true
		}
		for _, event := range message.events() {
			events = append(events, event)
			if first == "" && event.Type == harness.EventUser {
				first = event.Content
			}
		}
	}
	s.Mu.Lock()
	s.Events = events
	if s.FirstUserMsg == "" {
		s.FirstUserMsg = first
	}
	s.DoFullHistory = false
	s.Mu.Unlock()
	conn.Fetched, conn.LastUpdated = true, update
	_updateStats(title, update)
	return nil
}

func (a *Adapter) ListSessions(realFolder string) ([]harness.SessionLite, error) {
	result := make([]harness.SessionLite, 0)
	cursor := ""
	for {
		q := url.Values{"directory": {realFolder}, "archived": {"false"}, "limit": {"200"}}
		if cursor != "" {
			q.Set("cursor", cursor)
		}
		var page struct {
			Data []struct {
				ParentID string `json:"parentID"`
				ID       string `json:"id"`
				Title    string `json:"title"`
				Time     struct {
					Updated int64 `json:"updated"`
				} `json:"time"`
			} `json:"data"`
			Cursor struct {
				Next *string `json:"next"`
			} `json:"cursor"`
		}
		if err := (&Conn{}).call(http.MethodGet, "/api/session", q, nil, &page); err != nil {
			return nil, err
		}
		if len(page.Data) == 0 {
			break
		}
		for _, item := range page.Data {
			id := item.ID
			cacheMu.Lock()
			if local := toLocalIDs[id]; local != "" {
				id = local
			}
			cacheMu.Unlock()
			parentID := item.ParentID
			cacheMu.Lock()
			if local := toLocalIDs[parentID]; local != "" {
				parentID = local
			}
			cacheMu.Unlock()
			s := harness.SessionLite{ID: id, RealPath: realFolder, Harness: "opencode", Name: item.Title, LastTimestampMs: item.Time.Updated, ParentSessID: parentID}
			result = append(result, s)
		}
		if page.Cursor.Next == nil {
			break
		}
		cursor = *page.Cursor.Next
	}
	return result, nil
}
