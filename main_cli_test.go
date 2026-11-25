package main

import (
	"bytes"
	"testing"
	"time"
)

func TestCLIRingBufferKeepsNewestOutput(t *testing.T) {
	a := &app{config: appConfig{DefaultShell: "/bin/sh"}}
	session, err := a.getCLISession(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer session.pty.Close()
	if _, err := session.pty.WriteString("head -c 300000 /dev/zero | tr '\\0' a; echo MARKER_TAIL\r"); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		session.mu.Lock()
		tail := session.buffer
		if len(tail) > 256 {
			tail = tail[len(tail)-256:]
		}
		filled := len(session.buffer) == cliBufferSize && bytes.Contains(tail, []byte("MARKER_TAIL"))
		session.mu.Unlock()
		if filled {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("ring buffer did not cap at %d bytes with newest output", cliBufferSize)
}

func TestCleanupCLISessionRemovesAndCloses(t *testing.T) {
	cli := &cliSession{done: make(chan struct{})}
	a := &app{cliSessions: map[string]*cliSession{"/old": cli}}

	a.cleanupCLISession("/old")
	if _, ok := a.cliSessions["/old"]; ok {
		t.Fatal("CLI session remained cached")
	}
	cli.mu.Lock()
	closed := cli.closed
	cli.mu.Unlock()
	if !closed {
		t.Fatal("CLI session was not closed")
	}
	select {
	case <-cli.done:
	default:
		t.Fatal("CLI session was not signaled closed")
	}
}
