package harness

import (
	"errors"
	"strings"
	"sync"
)

// === everything is building up to Session
type EventType string

const (
	EventUser      EventType = "user"
	EventAssistant EventType = "assistant"
	EventThinking  EventType = "thinking"
	EventToolStart EventType = "tool_start"
	EventToolEnd   EventType = "tool_end"
	EventError     EventType = "error"
)

type Event struct {
	Type        EventType `json:"type"`
	Content     string    `json:"content,omitempty"`
	Images      []string  `json:"images,omitempty"` // data URIs
	ToolName    string    `json:"tool_name,omitempty"`
	ToolCallID  string    `json:"tool_call_id,omitempty"`
	ToolFailed  bool      `json:"tool_failed,omitempty"`
	TimestampMs int64     `json:"timestamp,omitempty"`
	Model       string    `json:"model,omitempty"`
}

type Model struct {
	Harness   string   `json:"harness"`
	ID        string   `json:"id"`
	Reasoning []string `json:"reasoning"`
}

type SessionLite struct {
	ID              string
	RealPath        string
	Harness         string
	Name            string
	FirstUserMsg    string
	LastTimestampMs int64
	ParentSessID    string
}
type Session struct {
	SessionLite
	Events     []Event
	Mu         sync.Mutex
	OpMu       sync.Mutex
	IsDeleted  bool
	IsDeleting bool
	IsStopping bool

	BusyMsg bool
	BusyCli bool

	EmptyPollCnt     int
	DoFullHistory    bool
	DoneFullHistory  chan struct{}
	NeedsFirstPrompt bool
	QueuedMsg        string
	QueuedImages     []string
	SendingMsg       string
	SendingImages    []string

	PromptModel     *Model
	PromptReasoning string

	// stats
	FullInputToken  int64
	FullOutputToken int64
	CacheReadToken  int64
	CacheWriteToken int64
	FullCost        float64
	ContextTokens   int64
	ContextWindow   int64
}

func (s *Session) GetIsBusy() bool {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	return s.BusyMsg || s.BusyCli || s.QueuedMsg != ""
}

// === then Adapter
type Adapter interface {
	// Ready checks service availability before the adapter is registered.
	Ready() error
	// Connect() should be cheap if already connected
	// Connect() should work for creating new sessions as well
	Connect(s *Session, doForceReconnect bool) error
	GetModels() (map[string]Model, error)
	ListSessions(realFolder string) ([]SessionLite, error)
	CanDelete() bool
	Delete(*Session) error
	SendPrompt(s *Session) error
	// Poll() refreshes the session title/first user message, stats, events, and other state.
	// Poll() should (if possible) do partial msg read, unless s.DoFullHistory=true, then do full read
	Poll(s *Session) error
	Stop(s *Session) error
	CleanUp(*Session) error
}

type DefaultAdapter struct{}

var _ Adapter = DefaultAdapter{}

func (DefaultAdapter) Ready() error { return errors.New("DefaultAdapter.Ready is not implemented") }
func (DefaultAdapter) Connect(*Session, bool) error {
	return errors.New("DefaultAdapter.Connect is not implemented")
}
func (DefaultAdapter) GetModels() (map[string]Model, error) {
	return nil, errors.New("DefaultAdapter.GetModels is not implemented")
}
func (DefaultAdapter) ListSessions(string) ([]SessionLite, error) {
	return nil, errors.New("DefaultAdapter.ListSessions is not implemented")
}
func (DefaultAdapter) SendPrompt(*Session) error {
	return errors.New("DefaultAdapter.SendPrompt is not implemented")
}
func (DefaultAdapter) Poll(*Session) error {
	return errors.New("DefaultAdapter.Poll is not implemented")
}
func (DefaultAdapter) Stop(*Session) error {
	return errors.New("DefaultAdapter.Stop is not implemented")
}
func (DefaultAdapter) CleanUp(*Session) error { return nil }
func (DefaultAdapter) CanDelete() bool        { return false }
func (DefaultAdapter) Delete(*Session) error {
	return errors.New("session delete is not supported")
}

// === helper fns
type Image struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
}

func ParseImage(uri string) (Image, error) {
	header, data, ok := strings.Cut(strings.TrimPrefix(uri, "data:"), ",")
	if !strings.HasPrefix(uri, "data:") || !ok || !strings.HasSuffix(header, ";base64") {
		return Image{}, errors.New("invalid image data URI")
	}
	mimeType := strings.TrimSuffix(header, ";base64")
	if !strings.HasPrefix(mimeType, "image/") || len(mimeType) == len("image/") {
		return Image{}, errors.New("invalid image MIME type")
	}
	return Image{Type: "image", Data: data, MimeType: mimeType}, nil
}

func DataURI(mimeType, data string) string {
	// for formattin links for inline imgs
	if data == "" {
		return ""
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return "data:" + mimeType + ";base64," + data
}
