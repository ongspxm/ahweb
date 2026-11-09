package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

func TestCLIWebsocketReusesSessionReplaysOutputAndResizes(t *testing.T) {
	folder := t.TempDir()
	a := &app{config: appConfig{DefaultShell: "/bin/sh"}}
	server := httptest.NewServer(http.HandlerFunc(a.handleCLI))
	defer server.Close()
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/cli"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	connect := func(offset ...int64) *websocket.Conn {
		t.Helper()
		query := url.Values{"folder": {folder}}
		if len(offset) > 0 {
			query.Set("offset", strconv.FormatInt(offset[0], 10))
		}
		conn, _, err := websocket.Dial(ctx, wsURL+"?"+query.Encode(), nil)
		if err != nil {
			t.Fatal(err)
		}
		return conn
	}
	first := connect()
	defer first.Close(websocket.StatusNormalClosure, "")
	if err := first.Write(ctx, websocket.MessageBinary, []byte("printf 'CLI_OUTPUT_MARKER\\n'\r")); err != nil {
		t.Fatal(err)
	}
	readUntil := func(conn *websocket.Conn, marker string) []byte {
		t.Helper()
		var output []byte
		for !bytes.Contains(output, []byte(marker)) {
			_, data, err := conn.Read(ctx)
			if err != nil {
				t.Fatalf("read terminal output: %v", err)
			}
			output = append(output, data...)
		}
		return output
	}
	readUntil(first, "CLI_OUTPUT_MARKER")

	second := connect()
	defer second.Close(websocket.StatusNormalClosure, "")
	readUntil(second, "CLI_OUTPUT_MARKER")
	a.cliMu.Lock()
	a.cliSessions[folder].mu.Lock()
	offset := a.cliSessions[folder].outputOffset
	a.cliSessions[folder].mu.Unlock()
	a.cliMu.Unlock()
	third := connect(offset)
	defer third.Close(websocket.StatusNormalClosure, "")
	kind, data, err := third.Read(ctx)
	if err != nil || kind != websocket.MessageText || !bytes.Contains(data, []byte(`"offset":`+strconv.FormatInt(offset, 10))) {
		t.Fatalf("expected resume offset %d, got %q (err %v)", offset, data, err)
	}
	if err := third.Write(ctx, websocket.MessageBinary, []byte("printf 'NEXT_MARKER\\n'\r")); err != nil {
		t.Fatal(err)
	}
	if output := readUntil(third, "NEXT_MARKER"); bytes.Contains(output, []byte("CLI_OUTPUT_MARKER")) {
		t.Fatal("resume replayed output already seen")
	}
	if err := second.Write(ctx, websocket.MessageText, []byte(`{"type":"resize","cols":91,"rows":37}`)); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(time.Second)
	resized := false
	for time.Now().Before(deadline) {
		a.cliMu.Lock()
		session := a.cliSessions[folder]
		a.cliMu.Unlock()
		if session != nil {
			rows, cols, err := pty.Getsize(session.pty)
			if err == nil && cols == 91 && rows == 37 {
				resized = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !resized {
		t.Fatal("PTY size did not update from websocket resize")
	}
	if err := first.Write(ctx, websocket.MessageBinary, []byte("exit\r")); err != nil {
		t.Fatal(err)
	}
	a.cliMu.Lock()
	session := a.cliSessions[folder]
	a.cliMu.Unlock()
	select {
	case <-session.done:
	case <-ctx.Done():
		t.Fatal("shell did not exit")
	}
}

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
