package websockethub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrUnauthorized              = errors.New("websocket subscription is unauthorized")
	ErrSlowClient                = errors.New("websocket client is too slow")
	ErrWorkspaceCapacityExceeded = errors.New("websocket workspace session capacity exceeded")
	ErrSubscriptionTooBroad      = errors.New("websocket subscription exceeds device filter capacity")
	ErrCursorExpired             = errors.New("websocket recovery cursor expired")
	ErrRecoveryUnavailable       = errors.New("websocket recovery provider unavailable")
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
	QueueSize               int
	MaxSessionsPerWorkspace int
	MaxDeviceFilters        int
	EventDedupeCapacity     int
	Authorizer              Authorizer
	Recovery                RecoveryProvider
	CapacityObserver        CapacityObserver
}

// CapacityObserver receives low-cardinality capacity outcomes. It keeps the
// Hub observable without coupling it to a metrics implementation.
type CapacityObserver interface {
	RecordCapacityEvent(component, outcome string)
}

type Stats struct {
	ActiveSessions        int
	RejectedConnections   uint64
	RejectedSubscriptions uint64
	SlowClientDisconnects uint64
	TelemetryCoalesced    uint64
	DuplicateEvents       uint64
}

type Hub struct {
	config                Config
	mu                    sync.RWMutex
	sessions              map[*Session]struct{}
	closed                bool
	rejectedConnections   atomic.Uint64
	rejectedSubscriptions atomic.Uint64
	slowClientDisconnects atomic.Uint64
	telemetryCoalesced    atomic.Uint64
	duplicateEvents       atomic.Uint64
}

func New(config Config) (*Hub, error) {
	if config.QueueSize < 1 {
		return nil, errors.New("websocket queue size must be positive")
	}
	if config.MaxSessionsPerWorkspace == 0 {
		config.MaxSessionsPerWorkspace = 64
	}
	if config.MaxDeviceFilters == 0 {
		config.MaxDeviceFilters = 100
	}
	if config.EventDedupeCapacity == 0 {
		config.EventDedupeCapacity = 2048
	}
	if config.MaxSessionsPerWorkspace < 1 || config.MaxDeviceFilters < 1 || config.EventDedupeCapacity < 1 {
		return nil, errors.New("websocket capacity limits must be positive")
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
	if h.workspaceSessionCountLocked(workspaceID) >= h.config.MaxSessionsPerWorkspace {
		h.rejectedConnections.Add(1)
		h.recordCapacity("websocket", "workspace_session_limit")
		return nil, ErrWorkspaceCapacityExceeded
	}
	session := &Session{
		hub: h, principal: principal, workspaceID: workspaceID, transport: transport,
		queue: make(chan Event, h.config.QueueSize), pendingTelemetry: make(map[string]Event),
		seenEvents: make(map[string]struct{}, h.config.EventDedupeCapacity),
		done:       make(chan struct{}),
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
			h.slowClientDisconnects.Add(1)
			h.recordCapacity("websocket", "slow_client_disconnect")
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

func (h *Hub) Stats() Stats {
	h.mu.RLock()
	activeSessions := len(h.sessions)
	h.mu.RUnlock()
	return Stats{
		ActiveSessions:        activeSessions,
		RejectedConnections:   h.rejectedConnections.Load(),
		RejectedSubscriptions: h.rejectedSubscriptions.Load(),
		SlowClientDisconnects: h.slowClientDisconnects.Load(),
		TelemetryCoalesced:    h.telemetryCoalesced.Load(),
		DuplicateEvents:       h.duplicateEvents.Load(),
	}
}

func (h *Hub) workspaceSessionCountLocked(workspaceID string) int {
	count := 0
	for session := range h.sessions {
		if session.workspaceID == workspaceID {
			count++
		}
	}
	return count
}

func (h *Hub) recordCapacity(component, outcome string) {
	if h.config.CapacityObserver != nil {
		h.config.CapacityObserver.RecordCapacityEvent(component, outcome)
	}
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
	seenEvents       map[string]struct{}
	seenOrder        []string
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
	if len(request.DeviceIDs) > s.hub.config.MaxDeviceFilters {
		s.hub.rejectedSubscriptions.Add(1)
		s.hub.recordCapacity("websocket", "device_filter_limit")
		return ErrSubscriptionTooBroad
	}
	if err := s.hub.config.Authorizer.Authorize(ctx, s.principal, s.workspaceID, request); err != nil {
		return err
	}
	s.mu.Lock()
	s.subscription = request
	s.mu.Unlock()

	readyAt := time.Now().UTC()
	readyData, _ := json.Marshal(map[string]any{"workspace_id": s.workspaceID, "cursor": request.Cursor})
	if err := s.enqueue(Event{
		EventID: fmt.Sprintf("session-ready-%s-%d", s.principal.Subject, readyAt.UnixNano()), Type: "session.ready",
		SchemaVersion: "1.0", WorkspaceID: s.workspaceID, OccurredAt: readyAt, Data: readyData,
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
	s.hub.remove(s)
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
	if event.EventID != "" {
		if _, seen := s.seenEvents[event.EventID]; seen {
			s.hub.duplicateEvents.Add(1)
			s.hub.recordCapacity("websocket", "duplicate_event")
			return nil
		}
	}
	select {
	case s.queue <- event:
		s.rememberEvent(event.EventID)
		return nil
	default:
		if isTelemetry(event) {
			key := event.AggregateID
			if key == "" {
				key = event.EventID
			}
			s.pendingTelemetry[key] = event
			s.rememberEvent(event.EventID)
			s.hub.telemetryCoalesced.Add(1)
			s.hub.recordCapacity("websocket", "telemetry_coalesced")
			return nil
		}
		return ErrSlowClient
	}
}

func (s *Session) rememberEvent(eventID string) {
	if eventID == "" {
		return
	}
	s.seenEvents[eventID] = struct{}{}
	s.seenOrder = append(s.seenOrder, eventID)
	if len(s.seenOrder) > s.hub.config.EventDedupeCapacity {
		oldest := s.seenOrder[0]
		delete(s.seenEvents, oldest)
		s.seenOrder = s.seenOrder[1:]
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
				s.hub.remove(s)
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
