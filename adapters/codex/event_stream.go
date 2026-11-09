package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/ongspxm/ahweb/harness"
)

type cachedEvent struct {
	method   string
	params   json.RawMessage
	sequence uint64
}

type rpcReply struct {
	ID       json.RawMessage `json:"id"`
	Result   json.RawMessage `json:"result"`
	sequence uint64          `json:"-"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type rpcConn struct {
	cmd      *exec.Cmd
	in       io.WriteCloser
	write    sync.Mutex
	mu       sync.Mutex
	nextID   int
	sequence uint64
	wait     map[string]chan rpcReply
	dead     chan struct{}
	err      error
	threads  map[string]*conn
}

var (
	rpcMu      sync.Mutex
	processRPC *rpcConn
)

// oni 1 main entry points: openStream()
func openStream() (*conn, error) {
	rpcMu.Lock()
	defer rpcMu.Unlock()
	if processRPC != nil {
		if processRPC.status() == nil {
			return &conn{eventRefs: map[string]itemEventRef{}}, nil
		}
		processRPC = nil
	}

	// startup the codex connection
	if _, err := exec.LookPath("codex"); err != nil {
		return nil, fmt.Errorf("Codex CLI is not installed: %w", err)
	}
	cmd := exec.Command("codex", "--dangerously-bypass-approvals-and-sandbox", "app-server", "--stdio")
	cwd, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not find user home: %w", err)
	}
	cmd.Dir = cwd
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Codex input: %w", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("could not open Codex output: %w", err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("could not start Codex App Server: %w", err)
	}
	rpc := &rpcConn{cmd: cmd, in: in, wait: map[string]chan rpcReply{}, dead: make(chan struct{}), threads: map[string]*conn{}}

	// loop for reading input, will keep adding to cachedEvent
	go func() {
		scanner := bufio.NewScanner(out)
		scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)
		for scanner.Scan() {
			var msg map[string]json.RawMessage
			if json.Unmarshal(scanner.Bytes(), &msg) != nil {
				continue
			}
			rpc.mu.Lock()
			rpc.sequence++
			sequence := rpc.sequence
			rpc.mu.Unlock()
			rawMethod, hasMethod := msg["method"]
			if !hasMethod {
				var reply rpcReply
				if json.Unmarshal(scanner.Bytes(), &reply) != nil || len(reply.ID) == 0 {
					continue
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
				continue
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
				continue
			}
			if c != nil {
				c.mu.Lock()
				c.eventCache = append(c.eventCache, cachedEvent{method: name, params: append(json.RawMessage(nil), msg["params"]...), sequence: sequence})
				c.mu.Unlock()
			}
		}
		waitErr := cmd.Wait()
		message := strings.TrimSpace(stderr.String())
		rpc.mu.Lock()
		switch {
		case scanner.Err() != nil:
			rpc.err = fmt.Errorf("could not read Codex App Server: %w", scanner.Err())
		case waitErr != nil:
			if message == "" {
				message = waitErr.Error()
			}
			rpc.err = fmt.Errorf("Codex App Server exited: %s", message)
		default:
			rpc.err = errors.New("Codex App Server exited")
		}
		for _, ch := range rpc.wait {
			ch <- rpcReply{Error: &struct {
				Message string `json:"message"`
			}{rpc.err.Error()}}
		}
		rpc.wait = map[string]chan rpcReply{}
		close(rpc.dead)
		rpc.mu.Unlock()
	}()

	// initialize the connection and kill if not up
	if _, _, err := rpc.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "ahweb", "title": "ahweb", "version": "1"}}); err != nil {
		_ = in.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, err
	}
	if err := rpc.send(map[string]any{"jsonrpc": "2.0", "method": "initialized", "params": map[string]any{}}); err != nil {
		_ = in.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, err
	}
	processRPC = rpc
	return &conn{eventRefs: map[string]itemEventRef{}}, nil
}

// === rpcConn and conn fns
func (rpc *rpcConn) status() error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	return rpc.err
}

func (rpc *rpcConn) send(msg map[string]any) error {
	rpc.write.Lock()
	defer rpc.write.Unlock()
	return json.NewEncoder(rpc.in).Encode(msg)
}

func (rpc *rpcConn) call(method string, params any) (json.RawMessage, uint64, error) {
	rpc.mu.Lock()
	if rpc.err != nil {
		err := rpc.err
		rpc.mu.Unlock()
		return nil, 0, err
	}
	rpc.nextID++
	id := rpc.nextID
	idRaw, _ := json.Marshal(id)
	ch := make(chan rpcReply, 1)
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
		if err := rpc.status(); err != nil {
			return nil, 0, err
		}
		return nil, 0, errors.New("Codex App Server exited")
	case <-time.After(30 * time.Second):
		rpc.mu.Lock()
		delete(rpc.wait, string(idRaw))
		rpc.mu.Unlock()
		return nil, 0, fmt.Errorf("Codex %s timed out", method)
	}
}
