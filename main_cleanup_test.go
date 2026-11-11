package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type cleanupTestAdapter struct {
	harness.DefaultAdapter
	sessions []harness.SessionLite
	calls    int
}

func (a *cleanupTestAdapter) ListSessions(string) ([]harness.SessionLite, error) {
	return a.sessions, nil
}
func (*cleanupTestAdapter) CanDelete() bool { return true }
func (a *cleanupTestAdapter) Delete(_ *harness.Session) error {
	a.calls++
	return nil
}

func TestCleanupPreviewCountsOldSessions(t *testing.T) {
	folder := t.TempDir()
	now := time.Now().UnixMilli()
	adapter := &cleanupTestAdapter{sessions: []harness.SessionLite{
		{ID: "recent", Harness: "test", RealPath: folder, LastTimestampMs: now - int64(2*24*time.Hour/time.Millisecond)},
		{ID: "delete", Harness: "test", RealPath: folder, LastTimestampMs: now - int64(10*24*time.Hour/time.Millisecond)},
	}}
	a := &app{
		sessions: make(map[string]map[sessionKey]*harness.Session),
		adapters: map[string]harness.Adapter{"test": adapter},
		config:   appConfig{GitRootRealDirs: folder},
	}
	q := url.Values{"days": {"4"}, "confirm": {"0"}}
	w := testRequest(a.handleCleanup, "POST", "/cleanup?"+q.Encode(), "")
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body)
	}
	var got struct {
		Planid string         `json:"planid"`
		Data   map[string]int `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Data[folder] != 1 {
		t.Fatalf("counts = %#v, want one delete", got.Data[folder])
	}
	if got.Planid == "" {
		t.Fatal("preview did not return a confirmation plan")
	}
	if adapter.calls != 0 {
		t.Fatalf("preview performed actions: %v", adapter.calls)
	}
}

func TestCleanupConfirmQueuesWorkAndRemovesSessionCache(t *testing.T) {
	folder := t.TempDir()
	adapter := &cleanupTestAdapter{sessions: []harness.SessionLite{{
		ID: "old", Harness: "test", RealPath: folder,
		LastTimestampMs: time.Now().Add(-5 * 24 * time.Hour).UnixMilli(),
	}}}
	a := &app{
		sessions: make(map[string]map[sessionKey]*harness.Session),
		adapters: map[string]harness.Adapter{"test": adapter},
		config:   appConfig{GitRootRealDirs: folder},
		pollJobs: make(chan pollJob, 1),
	}
	previewQ := url.Values{"days": {"4"}, "confirm": {"0"}}
	preview := testRequest(a.handleCleanup, "POST", "/cleanup?"+previewQ.Encode(), "")
	if preview.Code != 200 {
		t.Fatalf("preview status = %d, want 200: %s", preview.Code, preview.Body)
	}
	var previewBody struct {
		Planid string `json:"planid"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &previewBody); err != nil {
		t.Fatal(err)
	}
	plan := previewBody.Planid
	if plan == "" {
		t.Fatal("preview did not return a confirmation plan")
	}
	go func() { a.pollSession(<-a.pollJobs) }()
	q := url.Values{"days": {"4"}, "confirm": {"1"}, "plan": {plan}}
	w := testRequest(a.handleCleanup, "POST", "/cleanup?"+q.Encode(), "")
	if w.Code != 200 {
		t.Fatalf("status = %d, want 200: %s", w.Code, w.Body)
	}
	if adapter.calls != 1 {
		t.Fatalf("delete calls = %v, want one delete", adapter.calls)
	}
	if len(a.sessions) != 0 {
		t.Fatalf("cached sessions remain: %v", a.sessions)
	}
	var got struct {
		Data map[string]int `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Data[folder] != 1 {
		t.Fatalf("counts = %#v, want one delete", got.Data[folder])
	}
}

func TestCleanupRejectsNegativeDays(t *testing.T) {
	a := &app{}
	q := url.Values{"days": {"-1"}, "confirm": {"0"}}
	w := testRequest(a.handleCleanup, "POST", "/cleanup?"+q.Encode(), "")
	if w.Code != 400 {
		t.Fatalf("status = %d, want 400: %s", w.Code, w.Body)
	}
}

func TestCleanupApplyRejectsSessionChangedAfterPreview(t *testing.T) {
	folder := t.TempDir()
	adapter := &cleanupTestAdapter{sessions: []harness.SessionLite{{
		ID: "old", Harness: "test", RealPath: folder,
		LastTimestampMs: time.Now().Add(-5 * 24 * time.Hour).UnixMilli(),
	}}}
	a := &app{
		sessions: make(map[string]map[sessionKey]*harness.Session),
		adapters: map[string]harness.Adapter{"test": adapter},
		config:   appConfig{GitRootRealDirs: folder},
		pollJobs: make(chan pollJob, 1),
	}
	previewQ := url.Values{"days": {"4"}, "confirm": {"0"}}
	preview := testRequest(a.handleCleanup, "POST", "/cleanup?"+previewQ.Encode(), "")
	if preview.Code != 200 {
		t.Fatalf("preview status = %d: %s", preview.Code, preview.Body)
	}
	var previewBody struct {
		Planid string `json:"planid"`
	}
	if err := json.Unmarshal(preview.Body.Bytes(), &previewBody); err != nil {
		t.Fatal(err)
	}
	plan := previewBody.Planid
	s := a.sessions[folder][sessionKey{harness: "test", id: "old"}]
	s.Mu.Lock()
	s.LastTimestampMs = time.Now().UnixMilli()
	s.Mu.Unlock()
	go func() { a.pollSession(<-a.pollJobs) }()
	q := url.Values{"days": {"4"}, "confirm": {"1"}, "plan": {plan}}
	w := testRequest(a.handleCleanup, "POST", "/cleanup?"+q.Encode(), "")
	if w.Code != http.StatusMultiStatus {
		t.Fatalf("status = %d, want 207: %s", w.Code, w.Body)
	}
	if adapter.calls != 0 {
		t.Fatalf("changed session was deleted: %v", adapter.calls)
	}
	var got struct {
		Data  map[string]int `json:"data"`
		Error string         `json:"error"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Error == "" {
		t.Fatal("partial failure did not include an error")
	}
	if got.Data[folder] != 0 {
		t.Fatalf("counts = %#v, want no completed actions", got.Data[folder])
	}
	if a.sessions[folder][sessionKey{harness: "test", id: "old"}] != s {
		t.Fatal("failed session was removed from cache")
	}
}
