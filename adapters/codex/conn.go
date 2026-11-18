package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/ongspxm/ahweb/harness"
)

// === connection base

type connEvent struct {
	method   string
	params   json.RawMessage
	sequence uint64
}

type connReply struct {
	ID       json.RawMessage `json:"id"`
	Result   json.RawMessage `json:"result"`
	sequence uint64          `json:"-"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type connRPC struct {
	// write protects the ws so only one msg sender
	write sync.Mutex
	ws    *websocket.Conn

	mu       sync.Mutex
	nextID   int
	sequence uint64
	wait     map[string]chan connReply

	dead    chan struct{}
	err     error
	threads map[string]*SessionConn
}

func (c *SessionConn) drain() []connEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	events := c.eventCache
	c.eventCache = nil
	return events
}

func (c *SessionConn) undrain(events []connEvent) {
	if len(events) == 0 {
		return
	}
	c.mu.Lock()
	c.eventCache = append(events, c.eventCache...)
	c.mu.Unlock()
}

var (
	connRPCMu sync.Mutex
	connRPC0  *connRPC
)

// === daemon connection

// ensureRPC keeps one websocket connection to the persistent global app-server daemon.
func ensureRPC() error {
	connRPCMu.Lock()
	defer connRPCMu.Unlock()
	if connRPC0 != nil {
		if connRPC0.chkConnection() == nil {
			return nil
		}
		connRPC0.close()
		connRPC0 = nil
	}

	_connect := func() (*connRPC, error) {
		if _, err := exec.LookPath("codex"); err != nil {
			return nil, fmt.Errorf("Codex CLI is not installed: %w", err)
		}

		// the daemon is global: start it if needed and reuse it across ahweb restarts
		cmd := exec.Command("codex", "app-server", "daemon", "start")
		raw, err := cmd.Output()
		if err != nil {
			detail := strings.TrimSpace(string(raw))
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				if stderr := strings.TrimSpace(string(exitErr.Stderr)); stderr != "" {
					if detail != "" {
						detail += "\n"
					}
					detail += stderr
				}
			}
			return nil, fmt.Errorf("could not start Codex app-server daemon: %v: %s", err, detail)
		}
		var status struct {
			SocketPath string `json:"socketPath"`
		}
		if err := json.Unmarshal(raw, &status); err != nil {
			return nil, fmt.Errorf("invalid Codex daemon response: %w", err)
		}
		if status.SocketPath == "" {
			return nil, errors.New("Codex daemon did not report a socket path")
		}

		// app-server exposes a websocket on a unix socket
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		httpClient := &http.Client{Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", status.SocketPath)
			},
		}}
		ws, _, err := websocket.Dial(ctx, "ws://localhost/", &websocket.DialOptions{HTTPClient: httpClient})
		if err != nil {
			return nil, fmt.Errorf("could not connect to Codex app-server daemon: %w", err)
		}
		ws.SetReadLimit(10 << 20)
		rpc := &connRPC{ws: ws, wait: map[string]chan connReply{}, dead: make(chan struct{}), threads: map[string]*SessionConn{}}

		_handle := func(data []byte) {
			var msg map[string]json.RawMessage
			if json.Unmarshal(data, &msg) != nil {
				return
			}
			rpc.mu.Lock()
			rpc.sequence++
			sequence := rpc.sequence
			rpc.mu.Unlock()
			rawMethod, hasMethod := msg["method"]
			if !hasMethod {
				var reply connReply
				if json.Unmarshal(data, &reply) != nil || len(reply.ID) == 0 {
					return
				}
				reply.sequence = sequence
				key := string(reply.ID)
				rpc.mu.Lock()
				ch := rpc.wait[key]
				delete(rpc.wait, key)
				rpc.mu.Unlock()
				if ch != nil {
					ch <- reply
				}
				return
			}
			var name string
			_ = json.Unmarshal(rawMethod, &name)
			var params struct {
				ThreadID string `json:"threadId"`
			}
			_ = json.Unmarshal(msg["params"], &params)
			rpc.mu.Lock()
			c := rpc.threads[params.ThreadID]
			rpc.mu.Unlock()
			if rawID := msg["id"]; len(rawID) != 0 {
				content, code := "Codex requested unsupported input; ahweb cannot answer it.", -32601
				switch name {
				case "item/commandExecution/requestApproval", "item/fileChange/requestApproval":
					content = "Codex requested approval; ahweb does not support approvals, so the request was declined."
					_ = rpc.send(map[string]any{"jsonrpc": "2.0", "id": rawID, "result": map[string]any{"decision": "decline"}})
				case "item/permissions/requestApproval":
					content, code = "Codex requested extra permissions; ahweb cannot grant them.", -32000
					_ = rpc.send(map[string]any{"jsonrpc": "2.0", "id": rawID, "error": map[string]any{"code": code, "message": "ahweb does not grant additional Codex permissions"}})
				default:
					_ = rpc.send(map[string]any{"jsonrpc": "2.0", "id": rawID, "error": map[string]any{"code": code, "message": "ahweb cannot handle this Codex request"}})
				}
				if c != nil {
					c.mu.Lock()
					c.errors = append(c.errors, harness.Event{Type: harness.EventError, Content: content, TimestampMs: time.Now().UnixMilli()})
					c.mu.Unlock()
				}
				return
			}
			if c != nil {
				c.mu.Lock()
				c.eventCache = append(c.eventCache, connEvent{method: name, params: append(json.RawMessage(nil), msg["params"]...), sequence: sequence})
				c.mu.Unlock()
			}
		}
		go func() {
			defer rpc.ws.CloseNow()
			for {
				typ, data, err := rpc.ws.Read(context.Background())
				if err != nil {
					rpc.fail(fmt.Errorf("Codex app-server connection closed: %w", err))
					return
				}
				if typ == websocket.MessageText {
					_handle(data)
				}
			}
		}()

		if _, _, err := rpc.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "ahweb", "title": "ahweb", "version": "1"}}); err != nil {
			rpc.close()
			return nil, err
		}
		if err := rpc.send(map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}}); err != nil {
			rpc.close()
			return nil, err
		}
		return rpc, nil
	}

	var err error
	connRPC0, err = _connect()
	return err
}

// === RPC connection fns

func (rpc *connRPC) chkConnection() error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	return rpc.err
}

func (rpc *connRPC) fail(err error) {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	if rpc.err != nil {
		return
	}
	rpc.err = err
	for _, ch := range rpc.wait {
		ch <- connReply{Error: &struct {
			Message string `json:"message"`
		}{err.Error()}}
	}
	rpc.wait = map[string]chan connReply{}
	close(rpc.dead)
}

func (rpc *connRPC) close() {
	rpc.fail(errors.New("Codex app-server connection closed"))
	_ = rpc.ws.CloseNow()
}

// send is one-way, call also handles resp
func (rpc *connRPC) send(msg map[string]any) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	rpc.write.Lock()
	defer rpc.write.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := rpc.ws.Write(ctx, websocket.MessageText, data); err != nil {
		rpc.fail(fmt.Errorf("Codex app-server write failed: %w", err))
		return err
	}
	return nil
}

func (rpc *connRPC) call(method string, params any) (json.RawMessage, uint64, error) {
	rpc.mu.Lock()
	if rpc.err != nil {
		err := rpc.err
		rpc.mu.Unlock()
		return nil, 0, err
	}
	rpc.nextID++
	id := rpc.nextID
	idRaw, _ := json.Marshal(id)
	ch := make(chan connReply, 1)
	rpc.wait[string(idRaw)] = ch
	rpc.mu.Unlock()
	if err := rpc.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}); err != nil {
		rpc.mu.Lock()
		delete(rpc.wait, string(idRaw))
		rpc.mu.Unlock()
		return nil, 0, err
	}
	select {
	case reply := <-ch:
		if reply.Error != nil {
			return nil, reply.sequence, fmt.Errorf("Codex %s: %s", method, reply.Error.Message)
		}
		return reply.Result, reply.sequence, nil
	case <-rpc.dead:
		if err := rpc.chkConnection(); err != nil {
			return nil, 0, err
		}
		return nil, 0, errors.New("Codex app-server connection closed")
	case <-time.After(30 * time.Second):
		rpc.mu.Lock()
		delete(rpc.wait, string(idRaw))
		rpc.mu.Unlock()
		return nil, 0, fmt.Errorf("Codex %s timed out", method)
	}
}
