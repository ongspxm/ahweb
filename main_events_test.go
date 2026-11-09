package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type fullHistoryTestAdapter struct {
	harness.DefaultAdapter
	started chan struct{}
	release chan struct{}
}

func (a fullHistoryTestAdapter) Connect(*harness.Session, bool) error { return nil }
func (a fullHistoryTestAdapter) Poll(*harness.Session) error {
	close(a.started)
	<-a.release
	return nil
}

func TestEventsHandlerWaitsForFullHistoryPoll(t *testing.T) {
	folder := t.TempDir()
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "sess", Harness: "pi", RealPath: folder}, DoFullHistory: true, DoneFullHistory: make(chan struct{})}
	adapter := fullHistoryTestAdapter{started: make(chan struct{}), release: make(chan struct{})}
	a := &app{
		sessions: map[string]map[sessionKey]*harness.Session{folder: {{harness: "pi", id: "sess"}: s}},
		adapters: map[string]harness.Adapter{"pi": adapter},
	}
	a.lastOpened.times = make(map[string]map[string]int64)
	q := url.Values{"folder": {folder}, "sessid": {"sess"}, "harness": {"pi"}}
	res := httptest.NewRecorder()
	handlerDone := make(chan struct{})
	go func() {
		a.handleEvents(res, httptest.NewRequest(http.MethodGet, "/api/events?"+q.Encode(), nil))
		close(handlerDone)
	}()

	pollDone := make(chan struct{})
	go func() {
		var wg sync.WaitGroup
		wg.Add(1)
		a.pollSession(pollJob{session: s, done: &wg})
		wg.Wait()
		close(pollDone)
	}()
	select {
	case <-adapter.started:
	case <-time.After(time.Second):
		t.Fatal("full-history poll did not start")
	}
	select {
	case <-handlerDone:
		t.Fatal("handler returned before the full-history poll completed")
	default:
	}
	close(adapter.release)
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after the full-history poll completed")
	}
	<-pollDone
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
}
