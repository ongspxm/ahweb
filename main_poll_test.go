package main

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/ongspxm/ahweb/harness"
)

type fullPollFlagAdapter struct {
	harness.DefaultAdapter
	fullHistory bool
}

func (a *fullPollFlagAdapter) Connect(*harness.Session, bool) error { return nil }
func (a *fullPollFlagAdapter) Poll(s *harness.Session) error {
	s.Mu.Lock()
	a.fullHistory = s.DoFullHistory
	s.Mu.Unlock()
	return nil
}

func TestPollSessionForcesFullHistoryAfterEmptyPolls(t *testing.T) {
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "sess", Harness: "test"}, EmptyPollCnt: 1}
	adapter := &fullPollFlagAdapter{}
	a := &app{config: appConfig{NumEmptyPollBeforeCLIReset: 0}, adapters: map[string]harness.Adapter{"test": adapter}}
	var done sync.WaitGroup
	done.Add(1)
	a.pollSession(pollJob{session: s, done: &done})
	done.Wait()
	if !adapter.fullHistory {
		t.Fatal("Poll did not receive the forced full-history flag")
	}
}

// ordering check, between session fns like Connect(), Poll(), etc etc
type startupAdapter struct {
	harness.DefaultAdapter
	calls    []string
	failSend bool
}

func (a *startupAdapter) Connect(*harness.Session, bool) error {
	a.calls = append(a.calls, "connect")
	return nil
}
func (a *startupAdapter) Poll(*harness.Session) error { a.calls = append(a.calls, "poll"); return nil }
func (a *startupAdapter) CleanUp(*harness.Session) error {
	a.calls = append(a.calls, "cleanup")
	return nil
}
func (a *startupAdapter) SendPrompt(*harness.Session) error {
	a.calls = append(a.calls, "send")
	if a.failSend {
		return errors.New("send failed")
	}
	return nil
}

func TestFirstPromptBeforePoll(t *testing.T) {
	for _, tc := range []struct {
		name               string
		first, fail, empty bool
		want               string
	}{
		{"new", true, false, false, "connect,send,poll"},
		{"existing", false, false, false, "connect,poll,send"},
		{"new send failure", true, true, false, "connect,send"},
		{"existing send failure", false, true, false, "connect,poll,send"},
		{"waiting for queue", true, false, true, "connect"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &harness.Session{SessionLite: harness.SessionLite{ID: "sess", Harness: "test"}, NeedsFirstPrompt: tc.first, QueuedMsg: "hello"}
			if tc.empty {
				s.QueuedMsg = ""
			}
			adapter := &startupAdapter{failSend: tc.fail}
			a := &app{adapters: map[string]harness.Adapter{"test": adapter}}
			var done sync.WaitGroup
			done.Add(1)
			a.pollSession(pollJob{session: s, done: &done})
			if got := strings.Join(adapter.calls, ","); got != tc.want {
				t.Fatalf("calls = %s, want %s", got, tc.want)
			}
			if s.NeedsFirstPrompt != (tc.first && (tc.fail || tc.empty)) || (s.QueuedMsg != "") != tc.fail {
				t.Fatalf("unexpected prompt state: first=%v queued=%q", s.NeedsFirstPrompt, s.QueuedMsg)
			}
			if tc.first && tc.fail {
				adapter.failSend = false
				done.Add(1)
				a.pollSession(pollJob{session: s, done: &done})
				if got := strings.Join(adapter.calls, ","); got != "connect,send,connect,send,poll" || s.NeedsFirstPrompt || s.QueuedMsg != "" {
					t.Fatalf("first prompt retry failed: calls=%s first=%v queued=%q", got, s.NeedsFirstPrompt, s.QueuedMsg)
				}
			}
		})
	}
}
