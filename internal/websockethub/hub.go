package websockethub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrUnauthorized        = errors.New("websocket subscription is unauthorized")
	ErrSlowClient          = errors.New("websocket client is too slow")
	ErrCursorExpired       = errors.New("websocket recovery cursor expired")
	ErrRecoveryUnavailable = errors.New("websocket recovery provider unavailable")
)

type Principal struct {
	Subject      string
	WorkspaceIDs map[string]struct{}
}

type Authorizer interface {
	Authorize(context.Context, Principal, string, SubscriptionRequest) error
}

type WorkspaceAuthorizer struct{}

func (WorkspaceAuthorizer) Authorize(_ context.Context, principal Principal, workspaceID string, _ SubscriptionRequest) error {
	if workspaceID == "" {
		return ErrUnauthorized
	}
	if _, ok := principal.WorkspaceIDs[workspaceID]; !ok {
		return ErrUnauthorized
	}
	return nil
}

type SubscriptionRequest struct {
	WorkspaceID string   `json:"workspace_id,omitempty"`
	DeviceIDs   []string `json:"device_ids,omitempty"`
	Channels    []string `json:"channels,omitempty"`
	Cursor      string   `json:"cursor,omitempty"`
}

type Event struct {
	EventID       string          `json:"event_id"`
	Type          string          `json:"type"`
	SchemaVersion string          `json:"schema_version"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	AggregateID   string          `json:"aggregate_id,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Sequence      *int64          `json:"sequence,omitempty"`
	RequestID     string          `json:"request_id,omitempty"`
	Data          json.RawMessage `json:"data"`
}

type Transport interface {
	Write(context.Context, Event) error
	Close() error
}

type RecoveryProvider interface {
	Snapshot(context.Context, string, SubscriptionRequest) ([]Event, error)
	Replay(context.Context, string, string, SubscriptionRequest) ([]Event, error)
}

type Config struct {
	QueueSize  int
	Authorizer Authorizer
	Recovery   RecoveryProvider
}

type Hub struct {
	config   Config
	mu       sync.RWMutex
	sessions map[*Session]struct{}
	closed   bool
}

func New(config Config) (*Hub, error) {
	if config.QueueSize < 1 {
		return nil, errors.New("websocket queue size must be positive")
	}
	if config.Authorizer == nil {
		config.Authorizer = WorkspaceAuthorizer{}
	}
	return &Hub{config: config, sessions: make(map[*Session]struct{})}, nil
}

func (h *Hub) Connect(ctx context.Context, principal Principal, workspaceID string, transport Transport) (*Session, error) {
	if ctx == nil {
		return nil, errors.New("websocket context is required")
	}
	if transport == nil {
		return nil, errors.New("websocket transport is required")
	}
	if err := h.config.Authorizer.Authorize(ctx, principal, workspaceID, SubscriptionRequest{WorkspaceID: workspaceID}); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, errors.New("websocket hub is closed")
	}
	session := &Session{
		hub: h, principal: principal, workspaceID: workspaceID, transport: transport,
		queue: make(chan Event, h.config.QueueSize), pendingTelemetry: make(map[string]Event),
		done: make(chan struct{}),
	}
	h.sessions[session] = struct{}{}
	session.writerCtx, session.cancel = context.WithCancel(ctx)
	session.wg.Add(1)
	go session.writeLoop()
	return session, nil
}

func (h *Hub) Publish(event Event) {
	if event.WorkspaceID == "" {
		return
	}
	h.mu.RLock()
	sessions := make([]*Session, 0, len(h.sessions))
	for session := range h.sessions {
		sessions = append(sessions, session)
	}
	h.mu.RUnlock()
	for _, session := range sessions {
		if !session.matches(event) {
			continue
		}
		if err := session.enqueue(event); errors.Is(err, ErrSlowClient) {
			h.remove(session)
		}
	}
}

func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	sessions := make([]*Session, 0, len(h.sessions))
	for session := range h.sessions {
		sessions = append(sessions, session)
	}
	h.sessions = make(map[*Session]struct{})
	h.mu.Unlock()
	for _, session := range sessions {
		_ = session.Close()
	}
}

func (h *Hub) remove(session *Session) {
	h.mu.Lock()
	delete(h.sessions, session)
	h.mu.Unlock()
	session.closeOnce.Do(session.shutdown)
}

type Session struct {
	hub         *Hub
	principal   Principal
	workspaceID string
	transport   Transport

	mu               sync.RWMutex
	subscription     SubscriptionRequest
	queue            chan Event
	pendingTelemetry map[string]Event
	done             chan struct{}
	writerCtx        context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	closeOnce        sync.Once
}

func (s *Session) Subscribe(ctx context.Context, request SubscriptionRequest) error {
	if request.WorkspaceID != "" && request.WorkspaceID != s.workspaceID {
		return ErrUnauthorized
	}
	request.WorkspaceID = s.workspaceID
	if err := s.hub.config.Authorizer.Authorize(ctx, s.principal, s.workspaceID, request); err != nil {
		return err
	}
	s.mu.Lock()
	s.subscription = request
	s.mu.Unlock()

	readyData, _ := json.Marshal(map[string]any{"workspace_id": s.workspaceID, "cursor": request.Cursor})
	if err := s.enqueue(Event{
		EventID: "session-ready-" + s.principal.Subject, Type: "session.ready",
		SchemaVersion: "1.0", WorkspaceID: s.workspaceID, OccurredAt: time.Now().UTC(), Data: readyData,
	}); err != nil {
		return err
	}
	if s.hub.config.Recovery == nil {
		if request.Cursor != "" {
			return ErrRecoveryUnavailable
		}
		return nil
	}
	var (
		events []Event
		err    error
	)
	if request.Cursor == "" {
		events, err = s.hub.config.Recovery.Snapshot(ctx, s.workspaceID, request)
	} else {
		events, err = s.hub.config.Recovery.Replay(ctx, s.workspaceID, request.Cursor, request)
	}
	if err != nil {
		return err
	}
	for _, event := range events {
		if err := s.enqueue(event); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) Close() error {
	s.closeOnce.Do(func() {
		s.shutdown()
	})
	s.wg.Wait()
	return nil
}

func (s *Session) shutdown() {
	close(s.done)
	s.cancel()
	_ = s.transport.Close()
}

func (s *Session) Done() <-chan struct{} {
	return s.done
}

func (s *Session) enqueue(event Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	select {
	case <-s.done:
		return ErrClosedSession
	default:
	}
	select {
	case s.queue <- event:
		return nil
	default:
		if isTelemetry(event) {
			key := event.AggregateID
			if key == "" {
				key = event.EventID
			}
			s.pendingTelemetry[key] = event
			return nil
		}
		return ErrSlowClient
	}
}

func (s *Session) writeLoop() {
	defer s.wg.Done()
	for {
		select {
		case <-s.done:
			return
		case event := <-s.queue:
			if err := s.transport.Write(s.writerCtx, event); err != nil {
				s.closeOnce.Do(s.shutdown)
				return
			}
			s.flushPending()
		}
	}
}

func (s *Session) flushPending() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, event := range s.pendingTelemetry {
		select {
		case s.queue <- event:
			delete(s.pendingTelemetry, key)
		default:
			return
		}
	}
}

func (s *Session) matches(event Event) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.subscription.WorkspaceID != event.WorkspaceID {
		return false
	}
	if len(s.subscription.DeviceIDs) > 0 {
		found := false
		for _, deviceID := range s.subscription.DeviceIDs {
			if deviceID == event.AggregateID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(s.subscription.Channels) == 0 {
		return true
	}
	channel := eventChannel(event.Type)
	for _, allowed := range s.subscription.Channels {
		if allowed == channel {
			return true
		}
	}
	return channel == "session" || channel == "system"
}

func eventChannel(eventType string) string {
	switch {
	case len(eventType) >= 7 && eventType[:7] == "device.":
		if eventType == "device.telemetry" {
			return "telemetry"
		}
		return "state"
	case len(eventType) >= 6 && eventType[:6] == "alarm.":
		return "alarm"
	case len(eventType) >= 8 && eventType[:8] == "command.":
		return "command"
	case len(eventType) >= 8 && eventType[:8] == "session.":
		return "session"
	default:
		return "system"
	}
}

func isTelemetry(event Event) bool {
	return event.Type == "device.telemetry"
}

var ErrClosedSession = errors.New("websocket session is closed")

func (s *Session) String() string {
	return fmt.Sprintf("workspace=%s subject=%s", s.workspaceID, s.principal.Subject)
}
