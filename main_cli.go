package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

const cliBufferSize = 256 << 10

type cliSession struct {
	pty          *os.File
	cmd          *exec.Cmd
	writeMu      sync.Mutex
	mu           sync.Mutex
	buffer       []byte
	bufferStart  int64
	outputOffset int64
	clients      map[*cliClient]bool
	done         chan struct{}
	closed       bool
}

type cliClient struct {
	mu     sync.Mutex
	out    chan []byte
	done   chan struct{}
	closed bool
}

func (a *app) handleCLI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		apiMethodNotAllowed(w)
		return
	}
	folder, err := getAbsPath(r.URL.Query().Get("folder"))
	if err != nil {
		apiWriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	// setting up the connections
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(64 << 10)
	session, err := a.getCLISession(folder)
	if err != nil {
		_ = conn.Close(websocket.StatusInternalError, "could not start shell")
		return
	}
	client := &cliClient{out: make(chan []byte, 64), done: make(chan struct{})}
	offset, _ := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64)
	if offset < 0 {
		offset = 0
	}
	session.mu.Lock()
	if session.closed {
		session.mu.Unlock()
		_ = conn.Close(websocket.StatusTryAgainLater, "shell exited")
		return
	}
	if offset < session.bufferStart {
		offset = session.bufferStart
	}
	if offset > session.outputOffset {
		offset = session.outputOffset
	}
	replay := append([]byte(nil), session.buffer[offset-session.bufferStart:]...)
	session.clients[client] = true
	session.mu.Unlock()

	ctx, cancel := context.WithCancel(r.Context())
	defer func() {
		cancel()
		client.mu.Lock()
		if !client.closed {
			client.closed = true
			close(client.done)
		}
		client.mu.Unlock()
		session.mu.Lock()
		delete(session.clients, client)
		session.mu.Unlock()
		_ = conn.Close(websocket.StatusNormalClosure, "")
	}()

	// websocket admin fns
	go func() {
		defer cancel()
		meta, _ := json.Marshal(struct {
			Type   string `json:"type"`
			Offset int64  `json:"offset"`
		}{"offset", offset})
		if err := conn.Write(ctx, websocket.MessageText, meta); err != nil {
			return
		}
		if len(replay) > 0 {
			if err := conn.Write(ctx, websocket.MessageBinary, replay); err != nil {
				return
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case <-session.done:
				return
			case <-client.done:
				return
			case data := <-client.out:
				if err := conn.Write(ctx, websocket.MessageBinary, data); err != nil {
					return
				}
			}
		}
	}()

	// websocket msg fns
	for {
		kind, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		session.writeMu.Lock()
		session.mu.Lock()
		closed := session.closed
		session.mu.Unlock()
		if closed {
			session.writeMu.Unlock()
			return
		}
		if kind == websocket.MessageBinary {
			_, err = session.pty.Write(data)
		} else {
			var resize struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}
			if json.Unmarshal(data, &resize) == nil && resize.Type == "resize" && resize.Cols > 0 && resize.Cols <= 1000 && resize.Rows > 0 && resize.Rows <= 1000 {
				_ = pty.Setsize(session.pty, &pty.Winsize{Cols: uint16(resize.Cols), Rows: uint16(resize.Rows)})
			}
		}
		session.writeMu.Unlock()
		if err != nil {
			return
		}
	}
}

func (a *app) getCLISession(folder string) (*cliSession, error) {
	a.cliMu.Lock()
	defer a.cliMu.Unlock()
	if session := a.cliSessions[folder]; session != nil {
		session.mu.Lock()
		closed := session.closed
		session.mu.Unlock()
		if !closed {
			return session, nil
		}
	}
	a.mu.Lock()
	shell := strings.TrimSpace(a.config.DefaultShell)
	a.mu.Unlock()
	if shell == "" {
		shell = "bash"
	}
	cmd := exec.Command(shell)
	cmd.Dir = folder
	terminal, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	// actually set up the conenction
	_ = pty.Setsize(terminal, &pty.Winsize{Cols: 80, Rows: 24})
	session := &cliSession{pty: terminal, cmd: cmd, clients: make(map[*cliClient]bool), done: make(chan struct{})}
	if a.cliSessions == nil {
		a.cliSessions = make(map[string]*cliSession)
	}
	a.cliSessions[folder] = session
	go func() { _ = cmd.Wait() }()

	go func() {
		// read output, buffer it and send it out to everybody
		buf := make([]byte, 4096)
		for {
			n, err := terminal.Read(buf)
			if n > 0 {
				session.mu.Lock()
				session.buffer = append(session.buffer, buf[:n]...)
				session.outputOffset += int64(n)
				if over := len(session.buffer) - cliBufferSize; over > 0 {
					session.buffer = append(session.buffer[:0], session.buffer[over:]...)
					session.bufferStart += int64(over)
				}
				for client := range session.clients {
					client.mu.Lock()
					if client.closed {
						delete(session.clients, client)
						client.mu.Unlock()
						continue
					}
					select {
					case client.out <- append([]byte(nil), buf[:n]...):
					default:
						client.closed = true
						close(client.done)
						delete(session.clients, client)
					}
					client.mu.Unlock()
				}
				session.mu.Unlock()
			}
			if err != nil {
				session.mu.Lock()
				if !session.closed {
					session.closed = true
					close(session.done)
				}
				session.mu.Unlock()
				_ = terminal.Close()
				return
			}
		}
	}()
	return session, nil
}

func (a *app) cleanupCLISession(folder string) {
	a.cliMu.Lock()
	session := a.cliSessions[folder]
	delete(a.cliSessions, folder)
	a.cliMu.Unlock()
	if session == nil {
		return
	}
	session.mu.Lock()
	if !session.closed {
		session.closed = true
		close(session.done)
	}
	terminal, cmd := session.pty, session.cmd
	session.mu.Unlock()
	session.writeMu.Lock()
	defer session.writeMu.Unlock()
	if terminal != nil {
		_ = terminal.Close()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
