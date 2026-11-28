package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/ongspxm/ahweb/harness"
)

// === prompt and session handlers ===

func TestPromptHandlerAppendsNewPromptToExistingQueue(t *testing.T) {
	folder := t.TempDir()
	a := &app{adapters: map[string]harness.Adapter{"pi": harness.DefaultAdapter{}}, sessions: make(map[string]map[sessionKey]*harness.Session)}
	a.lastOpened.times = make(map[string]map[string]int64)
	prompt := func(sessID, value string) map[string]any {
		t.Helper()
		body, err := json.Marshal(map[string]string{"folder": folder, "sessid": sessID, "prompt": value, "harness": "pi"})
		if err != nil {
			t.Fatal(err)
		}
		res := testRequest(a.handlePrompt, http.MethodPost, "/api/prompt", string(body))
		if res.Code != http.StatusAccepted {
			t.Fatalf("prompt status = %d, want %d: %s", res.Code, http.StatusAccepted, res.Body.String())
		}
		var result map[string]any
		if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		return result
	}

	first := prompt("", " first ")
	second := prompt(first["sessid"].(string), "second")
	if got := second["queued"]; got != "first\nsecond" {
		t.Fatalf("queued = %q, want %q", got, "first\nsecond")
	}
}

func TestEventsHandlerReportsBusyWhenPromptIsQueued(t *testing.T) {
	folder := t.TempDir()
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "session", RealPath: folder, Harness: "pi"}, QueuedMsg: "waiting"}
	a := &app{adapters: map[string]harness.Adapter{"pi": harness.DefaultAdapter{}}, sessions: map[string]map[sessionKey]*harness.Session{folder: {{harness: s.Harness, id: s.ID}: s}}}
	a.lastOpened.times = make(map[string]map[string]int64)
	q := url.Values{"folder": {folder}, "sessid": {s.ID}, "harness": {s.Harness}}
	res := testRequest(a.handleEvents, http.MethodGet, "/api/events?"+q.Encode(), "")
	if res.Code != http.StatusOK {
		t.Fatalf("events status = %d, want %d: %s", res.Code, http.StatusOK, res.Body.String())
	}
	var result struct {
		Busy bool `json:"busy"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Busy {
		t.Fatal("busy = false with a queued prompt")
	}
}

func TestConfigHandlerRejectsUnknownConfigKey(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		t.Run(method, func(t *testing.T) {
			a := &app{}
			res := testRequest(a.handleConfig, method, "/api/config?"+url.Values{"key": {"not_a_config_key"}, "value": {"1"}}.Encode(), "")
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusBadRequest, res.Body.String())
			}
		})
	}
}

func TestConfigHandlerPersistsAndDeduplicatesDoneTours(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	a := &app{config: defaultConfig}
	post := func(key, value string) {
		t.Helper()
		q := url.Values{"key": {key}, "value": {value}}
		res := testRequest(a.handleConfig, http.MethodPost, "/api/config?"+q.Encode(), "")
		if res.Code != http.StatusOK {
			t.Fatalf("POST %s status = %d: %s", key, res.Code, res.Body.String())
		}
	}
	post("tour_done", "basic-tour")
	post("tour_done", "basic-tour")
	if len(a.config.ToursDone) != 1 || a.config.ToursDone[0] != "basic-tour" {
		t.Fatalf("config = %+v", a.config)
	}
}

// === queued prompt recovery ===

type queueFailureAdapter struct{ harness.DefaultAdapter }

func (queueFailureAdapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	s.QueuedMsg = "later"
	s.QueuedImages = []string{"later-image"}
	s.Mu.Unlock()
	return errors.New("send failed")
}

func TestSendQueuedRestoresQueuedMessagesAndImagesAfterFailure(t *testing.T) {
	s := &harness.Session{SessionLite: harness.SessionLite{Harness: "pi"}, BusyCli: true, QueuedMsg: "original", QueuedImages: []string{"original-image"}}
	(&app{}).sendQueued(s, queueFailureAdapter{})
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.QueuedMsg != "original\nlater" {
		t.Fatalf("queued message = %q, want original prompt followed by later prompt", s.QueuedMsg)
	}
	if got := strings.Join(s.QueuedImages, ","); got != "original-image,later-image" {
		t.Fatalf("queued images = %q, want original-image,later-image", got)
	}
	if !s.BusyCli || s.SendingMsg != "" || len(s.SendingImages) != 0 {
		t.Fatalf("send state not restored after failure: busy=%v sending=%q images=%v", s.BusyCli, s.SendingMsg, s.SendingImages)
	}
}

type promptRejectedAdapter struct{ harness.DefaultAdapter }

func (promptRejectedAdapter) SendPrompt(*harness.Session) error {
	return errors.Join(harness.ErrPromptRejected, errors.New("direct input disallowed"))
}

type promptRejectedWithQueuedAdapter struct{ harness.DefaultAdapter }

func (promptRejectedWithQueuedAdapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	s.QueuedMsg, s.QueuedImages = "later", []string{"later-image"}
	s.Mu.Unlock()
	return errors.Join(harness.ErrPromptRejected, errors.New("steer input rejected"))
}

func TestIfPromptRejectedShouldDropQueueAndErr(t *testing.T) {
	s := &harness.Session{
		SessionLite:      harness.SessionLite{Harness: "codex"},
		NeedsFirstPrompt: true,
		QueuedMsg:        "hello",
		QueuedImages:     []string{"img"},
	}
	(&app{}).sendQueued(s, promptRejectedAdapter{})
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.QueuedMsg != "" || len(s.QueuedImages) != 0 || s.SendingMsg != "" || len(s.SendingImages) != 0 || s.BusyCli || s.NeedsFirstPrompt {
		t.Fatalf("rejected prompt left dirty state: queued=%q images=%v sending=%q busy=%v first=%v", s.QueuedMsg, s.QueuedImages, s.SendingMsg, s.BusyCli, s.NeedsFirstPrompt)
	}
	if len(s.Events) != 1 || s.Events[0].Type != harness.EventError || !strings.Contains(s.Events[0].Content, "direct input disallowed") {
		t.Fatalf("events = %+v, want one error event", s.Events)
	}
}

func TestRejectedPromptShouldKeepLaterQueue(t *testing.T) {
	s := &harness.Session{SessionLite: harness.SessionLite{Harness: "prime"}, BusyCli: true, QueuedMsg: "reject this", QueuedImages: []string{"rejected-image"}}
	(&app{}).sendQueued(s, promptRejectedWithQueuedAdapter{})
	s.Mu.Lock()
	defer s.Mu.Unlock()
	if s.QueuedMsg != "later" || strings.Join(s.QueuedImages, ",") != "later-image" {
		t.Fatalf("later queue = %q %v, want later prompt and image", s.QueuedMsg, s.QueuedImages)
	}
	if !s.BusyCli || s.SendingMsg != "" || len(s.SendingImages) != 0 {
		t.Fatalf("send state not restored: busy=%v sending=%q images=%v", s.BusyCli, s.SendingMsg, s.SendingImages)
	}
}

// === handler request validation ===

func TestSessionHandlersRejectRequestsMissingHarness(t *testing.T) {
	folder := t.TempDir()
	query := url.Values{"folder": {folder}, "sessid": {"session"}}

	for _, endpoint := range []struct {
		name, method, target, body string
	}{
		{"events", http.MethodGet, "/api/events?" + query.Encode(), ""},
		{"stop", http.MethodPost, "/api/stop?" + query.Encode(), ""},
		{"prompt", http.MethodPost, "/api/prompt", `{"folder":"` + folder + `","prompt":"hello"}`},
	} {
		t.Run(endpoint.name, func(t *testing.T) {
			a := &app{}
			handler := map[string]http.HandlerFunc{"events": a.handleEvents, "stop": a.handleStop, "prompt": a.handlePrompt}[endpoint.name]
			res := testRequest(handler, endpoint.method, endpoint.target, endpoint.body)
			if res.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusBadRequest, res.Body.String())
			}
		})
	}
}

func TestReadOnlySessionShouldRejectPrompt(t *testing.T) {
	folder := t.TempDir()
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "child", RealPath: folder, Harness: "codex", ParentSessID: "parent", ReadOnly: true}}
	a := &app{
		adapters: map[string]harness.Adapter{"codex": harness.DefaultAdapter{}},
		sessions: map[string]map[sessionKey]*harness.Session{folder: {{harness: "codex", id: "child"}: s}},
	}
	body, err := json.Marshal(map[string]string{"folder": folder, "sessid": "child", "prompt": "hello", "harness": "codex"})
	if err != nil {
		t.Fatal(err)
	}
	res := testRequest(a.handlePrompt, http.MethodPost, "/api/prompt", string(body))
	if res.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d: %s", res.Code, http.StatusConflict, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "read-only") || s.QueuedMsg != "" {
		t.Fatalf("unexpected response %s or queued=%q", res.Body.String(), s.QueuedMsg)
	}
}

// === session listing and polling ===

type listedSessionsAdapter struct {
	harness.DefaultAdapter
	listed []harness.SessionLite
}

func (a listedSessionsAdapter) ListSessions(string) ([]harness.SessionLite, error) {
	return a.listed, nil
}

func (listedSessionsAdapter) Connect(*harness.Session, bool) error { return nil }
func (listedSessionsAdapter) Poll(*harness.Session) error          { return nil }

func TestSessionListUsesFirstUserMsgAsTitle(t *testing.T) {
	folder := t.TempDir()
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "session", Harness: "pi", RealPath: folder, FirstUserMsg: "  first prompt  "}}
	a := &app{sessions: map[string]map[sessionKey]*harness.Session{folder: {{harness: "pi", id: s.ID}: s}}}
	r := testRequest(a.handleList, http.MethodGet, "/api/list?folder="+url.QueryEscape(folder), "")
	if r.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", r.Code, r.Body)
	}
	var result struct {
		Sessions []struct {
			Name string `json:"name"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 1 || result.Sessions[0].Name != "first prompt" {
		t.Fatalf("session title = %+v, want first prompt", result.Sessions)
	}
	if got := sessionName(harness.SessionLite{FirstUserMsg: strings.Repeat("x", 141)}); len([]rune(got)) != 140 {
		t.Fatalf("title length = %d, want 140", len([]rune(got)))
	}
	if got := sessionName(harness.SessionLite{}); got != "" {
		t.Fatalf("empty title = %q, want empty", got)
	}
}
func TestSessionListingKeepsSameIDSeparateAcrossHarnesses(t *testing.T) {
	folder := t.TempDir()
	id := "shared-id"
	a := &app{sessions: make(map[string]map[sessionKey]*harness.Session), adapters: map[string]harness.Adapter{
		"pi":    listedSessionsAdapter{listed: []harness.SessionLite{{ID: id, Harness: "pi", RealPath: folder}}},
		"prime": listedSessionsAdapter{listed: []harness.SessionLite{{ID: id, Harness: "prime", RealPath: folder}}},
	}}
	a.listFolderSessions(folder)
	if got := len(a.sessions[folder]); got != 2 {
		t.Fatalf("session count = %d, want 2", got)
	}
	for _, harnessName := range []string{"pi", "prime"} {
		s := a.sessionLocked(folder, harnessName, id)
		if s == nil || s.Harness != harnessName {
			t.Fatalf("session lookup for %q = %v", harnessName, s)
		}
	}
}

func TestPollingAndSessionRefreshPreserveSessionIdentity(t *testing.T) {
	folder := t.TempDir()
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "shared-id", Harness: "pi", RealPath: folder}}
	adp := listedSessionsAdapter{listed: []harness.SessionLite{{ID: s.ID, Harness: s.Harness, RealPath: folder}}}
	a := &app{sessions: map[string]map[sessionKey]*harness.Session{
		folder: {{harness: s.Harness, id: s.ID}: s},
	}, adapters: map[string]harness.Adapter{"pi": adp}}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 100 {
			a.listFolderSessions(folder)
		}
	}()
	go func() {
		defer wg.Done()
		for range 100 {
			var pollJobsDone sync.WaitGroup
			pollJobsDone.Add(1)
			a.pollSession(pollJob{session: s, pollJobsDone: &pollJobsDone})
			pollJobsDone.Wait()
		}
	}()
	wg.Wait()
	if s.ID != "shared-id" || s.Harness != "pi" {
		t.Fatalf("session identity changed: (%q, %q)", s.Harness, s.ID)
	}
}

// === model handler ===

type modelsAdapter struct {
	harness.DefaultAdapter
	models map[string]harness.Model
	err    error
}

func (a modelsAdapter) GetModels() (map[string]harness.Model, error) {
	return a.models, a.err
}

func TestModelsHandlerRefreshesHealthyAdaptersAndRemovesFailedAdapterCache(t *testing.T) {
	fresh := map[string]harness.Model{"new/model": {Harness: "good", ID: "new/model", Reasoning: []string{"high"}}}
	stale := map[string]harness.Model{"old/model": {Harness: "bad", ID: "old/model"}}
	unrelated := map[string]harness.Model{"keep/model": {Harness: "other", ID: "keep/model"}}
	a := &app{
		adapters: map[string]harness.Adapter{
			"good": modelsAdapter{models: fresh},
			"bad":  modelsAdapter{err: errors.New("unavailable")},
		},
		models: map[string]map[string]harness.Model{
			"good":  {"old/good": {ID: "old/good"}},
			"bad":   stale,
			"other": unrelated,
		},
	}

	testRequest(a.handleStatus, "GET", "/api/status", "")
	r := testRequest(a.handleModels, "GET", "/api/models", "")
	if r.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", r.Code, r.Body)
	}
	if _, ok := a.models["good"]["new/model"]; !ok || len(a.models["good"]) != 1 {
		t.Fatalf("good cache = %+v, want refreshed models", a.models["good"])
	}
	if _, ok := a.models["bad"]; ok {
		t.Fatalf("bad cache = %+v, want failed adapter removed", a.models["bad"])
	}
	if _, ok := a.models["other"]["keep/model"]; !ok {
		t.Fatalf("unrelated cache = %+v, want preserved", a.models["other"])
	}
	if got := r.Body.String(); !containsModelResponse(got, `"good"`, `"new/model"`) || containsModelResponse(got, `"bad"`) {
		t.Fatalf("partial response = %s", got)
	}
}

func containsModelResponse(body string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(body, value) {
			return false
		}
	}
	return true
}
