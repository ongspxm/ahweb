package prime

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// === connection base

type connBase struct {
	mu      sync.Mutex
	socket  net.Conn
	decoder *json.Decoder
	retryAt time.Time
}

func (d *connBase) ensureLocked() (bool, error) {
	if d.socket != nil {
		return false, nil
	}
	if time.Now().Before(d.retryAt) {
		return false, fmt.Errorf("prime daemon connection retry is delayed")
	}
	socket := strings.TrimSpace(os.Getenv("PRIME_AGENT_DAEMON_SOCKET"))
	if socket == "" {
		socket = filepath.Join(os.TempDir(), fmt.Sprintf("prime-agent-rust-%d", os.Getuid()), "daemon.sock")
	}

	// create socket and conn in
	if _, err := os.Stat(socket); err != nil {
		d.retryAt = time.Now().Add(5 * time.Second)
		return false, fmt.Errorf("%w: socket %s: %v", errDaemonUnavailable, socket, err)
	}
	sock, err := net.DialTimeout("unix", socket, 2*time.Second)
	if err != nil {
		d.retryAt = time.Now().Add(5 * time.Second)
		return false, fmt.Errorf("%w: %v", errDaemonUnavailable, err)
	}
	decoder := json.NewDecoder(sock)
	_ = sock.SetReadDeadline(time.Now().Add(5 * time.Second))

	// send harlow, make sure it works.
	var hello struct {
		Type     string `json:"type"`
		Protocol struct {
			Name    string `json:"name"`
			Version int    `json:"version"`
		} `json:"protocol"`
		SchemaID string `json:"schemaId"`
	}
	if err := decoder.Decode(&hello); err != nil {
		_ = sock.Close()
		d.retryAt = time.Now().Add(5 * time.Second)
		return false, err
	}
	_ = sock.SetReadDeadline(time.Time{})
	schemaPrefix := fmt.Sprintf("protocol-%d-schema-", daemonProtocolVersion)
	if hello.Type != "daemon_hello" || hello.Protocol.Name != daemonProtocolName || hello.Protocol.Version != daemonProtocolVersion || !strings.HasPrefix(hello.SchemaID, schemaPrefix) {
		_ = sock.Close()
		return false, fmt.Errorf("%w: protocol %s v%d schema %q", errDaemonIncompatible, hello.Protocol.Name, hello.Protocol.Version, hello.SchemaID)
	}
	d.socket, d.decoder, d.retryAt = sock, decoder, time.Time{}
	return true, nil
}

func (d *connBase) closeLocked() {
	if d.socket != nil {
		_ = d.socket.Close()
		d.socket, d.decoder, d.retryAt = nil, nil, time.Time{}
	}
}

func (d *connBase) isConnected() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.socket != nil
}

// === command connection

type connCmd struct {
	connBase
	requestSequence int
}

var connCmd0 connCmd

func (c *connCmd) Ready() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	fresh, err := c.ensureLocked()
	if fresh {
		c.requestSequence = 0
	}
	return err
}

func (c *connCmd) call(primeID, command string, params map[string]any, data any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.socket == nil {
		return errDaemonUnavailable
	}
	c.requestSequence++

	// form the req
	id := fmt.Sprintf("%s-%d", command, c.requestSequence)
	body := make(map[string]any, len(params)+3)
	body["type"], body["id"] = command, id
	for key, value := range params {
		body[key] = value
	}
	if primeID != "" {
		body["activeSessionId"] = primeID
	}
	envelope := map[string]any{
		"type": "command", "id": id,
		"protocol": map[string]any{"name": daemonProtocolName, "version": daemonProtocolVersion},
		"clientId": daemonClientID, "command": body,
	}

	// sending the req
	if err := json.NewEncoder(c.socket).Encode(envelope); err != nil {
		c.closeLocked()
		return fmt.Errorf("%w: could not send %s command: %v", errDaemonUnavailable, command, err)
	}
	_ = c.socket.SetReadDeadline(time.Now().Add(30 * time.Second))

	// reading the resp, only read the id that of my sent msg
	// we ok to skip msgs because call() got mu.lock
	for {
		var response struct {
			Type    string          `json:"type"`
			ID      string          `json:"id"`
			Success bool            `json:"success"`
			Error   string          `json:"error"`
			Data    json.RawMessage `json:"data"`
		}
		if err := c.decoder.Decode(&response); err != nil {
			c.closeLocked()
			return fmt.Errorf("%w: could not read %s response: %v", errDaemonUnavailable, command, err)
		}
		if response.Type == "daemon_closing" {
			c.closeLocked()
			return fmt.Errorf("%w: daemon closed the connection", errDaemonUnavailable)
		}
		if response.Type != "response" || response.ID != id {
			continue
		}
		if !response.Success {
			return errors.New(response.Error)
		}
		if data != nil && len(response.Data) > 0 && string(response.Data) != "null" {
			if err := json.Unmarshal(response.Data, data); err != nil {
				return fmt.Errorf("could not decode prime daemon %s response: %w", command, err)
			}
		}
		return nil
	}
}

// === event connection

type evtReply struct {
	success bool
	err     string
}

type connEvt struct {
	connBase
	commandSequence int

	// all these have ID as key
	// needsFull hints to Poll() to do fullHistory
	subs       map[string]bool
	events     map[string][]json.RawMessage
	needsFull  map[string]bool
	replyChans map[string]chan evtReply
}

var connEvt0 = connEvt{
	subs:       map[string]bool{},
	events:     map[string][]json.RawMessage{},
	needsFull:  map[string]bool{},
	replyChans: map[string]chan evtReply{},
}

func (e *connEvt) Ready() error {
	// when evt conn is up, we need to keep the long running evt loop going
	e.mu.Lock()
	defer e.mu.Unlock()
	fresh, err := e.ensureLocked()

	if fresh {
		sock, decoder := e.socket, e.decoder
		go func() {
			// handle closing of socket
			defer func() {
				_ = sock.Close()
				e.mu.Lock()
				if e.socket == sock {
					e.socket = nil
					for primeID := range e.subs {
						e.needsFull[primeID] = true
					}
					e.subs = make(map[string]bool)
					e.retryAt = time.Now().Add(5 * time.Second)
					for id, reply := range e.replyChans {
						reply <- evtReply{err: "Prime event connection closed"}
						delete(e.replyChans, id)
					}
				}
				e.mu.Unlock()
			}()

			// loop to read events
			for {
				var frame map[string]json.RawMessage
				if decoder.Decode(&frame) != nil {
					return
				}
				var kind, id string
				_ = json.Unmarshal(frame["type"], &kind)
				_ = json.Unmarshal(frame["id"], &id)
				if kind == "daemon_closing" {
					return
				}
				if kind == "response" {
					e.mu.Lock()
					if reply := e.replyChans[id]; reply != nil {
						var result evtReply
						_ = json.Unmarshal(frame["success"], &result.success)
						_ = json.Unmarshal(frame["error"], &result.err)
						reply <- result
					}
					e.mu.Unlock()
					continue
				}
				var primeID string
				_ = json.Unmarshal(frame["activeSessionId"], &primeID)
				if primeID == "" {
					continue
				}
				if kind != "session_event" {
					switch kind {
					case "session_replaced", "session_resynced", "session_closed", "session_snapshot_begin", "session_snapshot_chunk", "session_snapshot_end", "session_snapshot_failed":
						e.mu.Lock()
						e.needsFull[primeID] = true
						e.mu.Unlock()
					}
					continue
				}
				raw, err := json.Marshal(frame)
				if err != nil {
					continue
				}
				e.mu.Lock()
				events := e.events[primeID]
				if len(events) >= 256 {
					copy(events, events[1:])
					events = events[:255]
					e.needsFull[primeID] = true
				}
				e.events[primeID] = append(events, raw)
				e.mu.Unlock()
			}
		}()
	}
	return err
}

func (e *connEvt) isSub(primeID string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.subs[primeID]
}

func sub(c *SessionConn) error {
	if c.ActiveSessionID == "" {
		return nil
	}
	if err := connEvt0.Ready(); err != nil {
		return err
	}
	e, primeID := &connEvt0, c.ActiveSessionID
	e.mu.Lock()
	if e.subs[primeID] {
		e.mu.Unlock()
		return nil
	}
	if e.socket == nil {
		e.mu.Unlock()
		return fmt.Errorf("Prime event connection is unavailable")
	}
	e.commandSequence++

	// form the req to sub and send it
	id := fmt.Sprintf("attach-events-%d", e.commandSequence)
	sock := e.socket
	respChan := make(chan evtReply, 1)
	e.replyChans[id] = respChan
	clientID := fmt.Sprintf("%s-events-%s", daemonClientID, primeID)
	envelope := map[string]any{
		"type": "command", "id": id,
		"protocol": map[string]any{"name": daemonProtocolName, "version": daemonProtocolVersion},
		"clientId": daemonClientID + "-events",
		"command": map[string]any{
			"type": "attach", "id": id, "activeSessionId": primeID,
			"clientId": clientID, "capabilities": []string{"event_sequence"},
		},
	}
	err := json.NewEncoder(sock).Encode(envelope)
	e.mu.Unlock()

	// cleanup the state if got err
	_rmCmdReply := func() {
		e.mu.Lock()
		delete(e.replyChans, id)
		e.mu.Unlock()
	}
	if err != nil {
		_rmCmdReply()
		return err
	}
	var result evtReply
	select {
	case result = <-respChan:
	case <-time.After(30 * time.Second):
		_rmCmdReply()
		return fmt.Errorf("timed out attaching Prime event listener")
	}
	_rmCmdReply()
	if !result.success {
		return fmt.Errorf("prime daemon could not attach event listener: %s", result.err)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.socket != sock {
		return fmt.Errorf("Prime event connection closed during attach")
	}

	// setting up all the state
	e.subs[primeID] = true
	return nil
}

func unsub(c *SessionConn) {
	// param chks + early exit
	if c == nil || c.ActiveSessionID == "" {
		return
	}
	e, primeID := &connEvt0, c.ActiveSessionID
	e.mu.Lock()
	if e.socket == nil || !e.subs[primeID] {
		delete(e.subs, primeID)
		delete(e.events, primeID)
		delete(e.needsFull, primeID)
		e.mu.Unlock()
		return
	}
	e.commandSequence++

	// send the sub
	id := fmt.Sprintf("detach-events-%d", e.commandSequence)
	sock := e.socket
	reply := make(chan evtReply, 1)
	e.replyChans[id] = reply
	envelope := map[string]any{
		"type": "command", "id": id,
		"protocol": map[string]any{"name": daemonProtocolName, "version": daemonProtocolVersion},
		"clientId": daemonClientID + "-events",
		"command":  map[string]any{"type": "detach", "id": id, "activeSessionId": primeID},
	}
	err := json.NewEncoder(sock).Encode(envelope)
	e.mu.Unlock()
	if err == nil {
		select {
		case <-reply:
		case <-time.After(2 * time.Second):
		}
	}

	// finalize by closing the channel
	e.mu.Lock()
	delete(e.replyChans, id)
	if e.socket == sock {
		delete(e.subs, primeID)
		delete(e.events, primeID)
		delete(e.needsFull, primeID)
	}
	e.mu.Unlock()
}

func (e *connEvt) drainSub(primeID string) ([]json.RawMessage, bool) {
	// process events from the stream
	e.mu.Lock()
	defer e.mu.Unlock()
	events := e.events[primeID]
	needsFull := e.needsFull[primeID]
	delete(e.events, primeID)
	delete(e.needsFull, primeID)
	return events, needsFull
}

func (e *connEvt) undrainSub(primeID string, events []json.RawMessage, needsFull bool) {
	// put the events back, because processing failed.
	if len(events) == 0 && !needsFull {
		return
	}
	e.mu.Lock()
	queued := append(events, e.events[primeID]...)
	if len(queued) > 256 {
		queued = queued[len(queued)-256:]
		e.needsFull[primeID] = true
	}
	e.events[primeID] = queued
	if needsFull {
		e.needsFull[primeID] = true
	}
	e.mu.Unlock()
}
