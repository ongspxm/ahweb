package main

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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
	var pollJobsDone sync.WaitGroup
	pollJobsDone.Add(1)
	a.pollSession(pollJob{session: s, pollJobsDone: &pollJobsDone})
	pollJobsDone.Wait()
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
func (a *startupAdapter) Rewind(*harness.Session, string) error {
	a.calls = append(a.calls, "rewind")
	return nil
}
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

func TestRewindConnectsBeforeRewind(t *testing.T) {
	s := &harness.Session{SessionLite: harness.SessionLite{ID: "sess", Harness: "test"}}
	adapter := &startupAdapter{}
	a := &app{adapters: map[string]harness.Adapter{"test": adapter}}
	pollJobsDone := &sync.WaitGroup{}
	pollJobsDone.Add(1)
	result := make(chan error, 1)
	a.pollSession(pollJob{session: s, pollJobsDone: pollJobsDone, rewindResult: result, rewindTargetMsgID: "entry"})
	pollJobsDone.Wait()
	if err := <-result; err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(adapter.calls, ","); got != "connect,rewind,connect,poll" {
		t.Fatalf("calls = %s, want connect,rewind,connect,poll", got)
	}
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
			var pollJobsDone sync.WaitGroup
			pollJobsDone.Add(1)
			a.pollSession(pollJob{session: s, pollJobsDone: &pollJobsDone})
			pollJobsDone.Wait()
			if got := strings.Join(adapter.calls, ","); got != tc.want {
				t.Fatalf("calls = %s, want %s", got, tc.want)
			}
			if s.NeedsFirstPrompt != (tc.first && (tc.fail || tc.empty)) || (s.QueuedMsg != "") != tc.fail {
				t.Fatalf("unexpected prompt state: first=%v queued=%q", s.NeedsFirstPrompt, s.QueuedMsg)
			}
			if tc.first && tc.fail {
				adapter.failSend = false
				var pollJobsDone sync.WaitGroup
				pollJobsDone.Add(1)
				a.pollSession(pollJob{session: s, pollJobsDone: &pollJobsDone})
				pollJobsDone.Wait()
				if got := strings.Join(adapter.calls, ","); got != "connect,send,connect,send,poll" || s.NeedsFirstPrompt || s.QueuedMsg != "" {
					t.Fatalf("first prompt retry failed: calls=%s first=%v queued=%q", got, s.NeedsFirstPrompt, s.QueuedMsg)
				}
			}
		})
	}
}

// === test for steering / sending prompt
type autoSteerTestAdapter struct {
	harness.DefaultAdapter
	canSteer   bool
	busyAtSend []bool
}

func (a *autoSteerTestAdapter) CanSteer(bool) bool                 { return a.canSteer }
func (*autoSteerTestAdapter) Connect(*harness.Session, bool) error { return nil }
func (*autoSteerTestAdapter) Poll(*harness.Session) error          { return nil }
func (a *autoSteerTestAdapter) SendPrompt(s *harness.Session) error {
	s.Mu.Lock()
	a.busyAtSend = append(a.busyAtSend, s.BusyCli)
	s.BusyCli = true
	s.Mu.Unlock()
	return nil
}

func TestQueuedPromptShouldSendWhenIdleOrCanSteer(t *testing.T) {
	for _, tc := range []struct {
		name     string
		busyCli  bool
		busyMsg  bool
		canSteer bool
		wantSend bool
	}{
		{"idle", false, false, true, true},
		{"busy and can steer", true, false, true, true},
		{"busy without support", true, false, false, false},
		{"message busy but cli idle", false, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := &harness.Session{SessionLite: harness.SessionLite{ID: "sess", Harness: "test"}, BusyCli: tc.busyCli, QueuedMsg: "hello"}
			if tc.busyMsg {
				now := time.Now().UnixMilli()
				s.LastTimestampMs = now
				s.Events = []harness.Event{{Type: harness.EventUser, TimestampMs: now}}
			}
			adapter := &autoSteerTestAdapter{canSteer: tc.canSteer}
			a := &app{adapters: map[string]harness.Adapter{"test": adapter}}
			var pollJobsDone sync.WaitGroup
			pollJobsDone.Add(1)
			a.pollSession(pollJob{session: s, pollJobsDone: &pollJobsDone})
			pollJobsDone.Wait()
			if got := len(adapter.busyAtSend) != 0; got != tc.wantSend {
				t.Fatalf("sent = %v, want %v", got, tc.wantSend)
			}
			if tc.wantSend && adapter.busyAtSend[0] != tc.busyCli {
				t.Fatalf("BusyCli at SendPrompt = %v, want %v", adapter.busyAtSend[0], tc.busyCli)
			}
			if tc.wantSend != (s.QueuedMsg == "") {
				t.Fatalf("queued = %q, want cleared = %v", s.QueuedMsg, tc.wantSend)
			}
		})
	}
}
