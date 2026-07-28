package websockethub

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestHubAuthorizesWorkspaceAndFiltersEvents(t *testing.T) {
	hub, err := New(Config{QueueSize: 4})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	transport := newRecordingTransport()
	session, err := hub.Connect(context.Background(), Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}, "ws-1", transport)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer hub.Close()
	if err := session.Subscribe(context.Background(), SubscriptionRequest{Channels: []string{"alarm"}}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	hub.Publish(Event{EventID: "telemetry-1", Type: "device.telemetry", WorkspaceID: "ws-1", AggregateID: "device-1", OccurredAt: time.Now(), Data: []byte(`{}`)})
	hub.Publish(Event{EventID: "alarm-1", Type: "alarm.created", WorkspaceID: "ws-1", AggregateID: "device-1", OccurredAt: time.Now(), Data: []byte(`{}`)})
	waitForEvent(t, transport, "alarm.created")
	if _, err := hub.Connect(context.Background(), Principal{Subject: "other", WorkspaceIDs: map[string]struct{}{"ws-2": {}}}, "ws-1", newRecordingTransport()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("unauthorized workspace connection error = %v", err)
	}
}

func TestHubEnforcesWorkspaceAndSubscriptionCapacity(t *testing.T) {
	observer := &recordingCapacityObserver{}
	hub, err := New(Config{
		QueueSize:               4,
		MaxSessionsPerWorkspace: 1,
		MaxDeviceFilters:        2,
		CapacityObserver:        observer,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hub.Close()
	principal := Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}
	session, err := hub.Connect(context.Background(), principal, "ws-1", newRecordingTransport())
	if err != nil {
		t.Fatalf("first Connect() error = %v", err)
	}
	if _, err := hub.Connect(context.Background(), principal, "ws-1", newRecordingTransport()); !errors.Is(err, ErrWorkspaceCapacityExceeded) {
		t.Fatalf("second Connect() error = %v, want workspace capacity error", err)
	}
	if err := session.Subscribe(context.Background(), SubscriptionRequest{DeviceIDs: []string{"d-1", "d-2", "d-3"}}); !errors.Is(err, ErrSubscriptionTooBroad) {
		t.Fatalf("Subscribe() error = %v, want device filter capacity error", err)
	}
	stats := hub.Stats()
	if stats.ActiveSessions != 1 || stats.RejectedConnections != 1 || stats.RejectedSubscriptions != 1 {
		t.Fatalf("Stats() = %+v, want one active and one rejection of each kind", stats)
	}
	if !observer.has("websocket", "workspace_session_limit") || !observer.has("websocket", "device_filter_limit") {
		t.Fatalf("capacity events = %#v", observer.events)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	replacement, err := hub.Connect(context.Background(), principal, "ws-1", newRecordingTransport())
	if err != nil {
		t.Fatalf("Connect() after close error = %v", err)
	}
	if err := replacement.Close(); err != nil {
		t.Fatalf("replacement Close() error = %v", err)
	}
}

func TestSessionRecoveryUsesSnapshotOrCursorReplay(t *testing.T) {
	recovery := &fakeRecovery{
		snapshot: []Event{{EventID: "snapshot-1", Type: "device.state_changed", WorkspaceID: "ws-1", Data: []byte(`{}`)}},
		replay:   []Event{{EventID: "replay-1", Type: "command.updated", WorkspaceID: "ws-1", Data: []byte(`{}`)}},
	}
	hub, _ := New(Config{QueueSize: 8, Recovery: recovery})
	transport := newRecordingTransport()
	session, err := hub.Connect(context.Background(), Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}, "ws-1", transport)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer hub.Close()
	if err := session.Subscribe(context.Background(), SubscriptionRequest{}); err != nil {
		t.Fatalf("snapshot Subscribe() error = %v", err)
	}
	waitForEvent(t, transport, "device.state_changed")
	if err := session.Subscribe(context.Background(), SubscriptionRequest{Cursor: "cursor-1"}); err != nil {
		t.Fatalf("replay Subscribe() error = %v", err)
	}
	waitForEvent(t, transport, "command.updated")
	if recovery.snapshotCalls != 1 || recovery.replayCalls != 1 {
		t.Fatalf("recovery calls = snapshot:%d replay:%d", recovery.snapshotCalls, recovery.replayCalls)
	}
}

func TestSessionDeduplicatesOverlapBetweenRecoveryAndLiveEvents(t *testing.T) {
	event := Event{EventID: "alarm-1", Type: "alarm.created", WorkspaceID: "ws-1", Data: []byte(`{}`)}
	hub, err := New(Config{QueueSize: 8, Recovery: &fakeRecovery{snapshot: []Event{event}}})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer hub.Close()
	transport := newRecordingTransport()
	session, err := hub.Connect(context.Background(), Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}, "ws-1", transport)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := session.Subscribe(context.Background(), SubscriptionRequest{Channels: []string{"alarm"}}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	waitForEvent(t, transport, "alarm.created")
	hub.Publish(event)
	time.Sleep(10 * time.Millisecond)
	if got := countEvents(transport, "alarm.created"); got != 1 {
		t.Fatalf("alarm delivery count = %d, want one", got)
	}
	if hub.Stats().DuplicateEvents != 1 {
		t.Fatalf("hub stats = %+v", hub.Stats())
	}
}

func TestSlowClientCoalescesTelemetryAndClosesForDurableEvent(t *testing.T) {
	transport := newRecordingTransport()
	transport.block = make(chan struct{})
	hub, _ := New(Config{QueueSize: 1})
	session, err := hub.Connect(context.Background(), Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}, "ws-1", transport)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := session.Subscribe(context.Background(), SubscriptionRequest{}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	hub.Publish(Event{EventID: "telemetry-1", Type: "device.telemetry", WorkspaceID: "ws-1", AggregateID: "device-1", Data: []byte(`{"v":1}`)})
	hub.Publish(Event{EventID: "telemetry-2", Type: "device.telemetry", WorkspaceID: "ws-1", AggregateID: "device-1", Data: []byte(`{"v":2}`)})
	hub.Publish(Event{EventID: "alarm-1", Type: "alarm.created", WorkspaceID: "ws-1", AggregateID: "device-1", Data: []byte(`{}`)})
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("slow durable-event client was not closed")
	}
	close(transport.block)
}

func TestSessionWriteFailureDoesNotDeadlockClose(t *testing.T) {
	transport := newRecordingTransport()
	transport.writeErr = errors.New("connection reset")
	hub, _ := New(Config{QueueSize: 2})
	session, err := hub.Connect(context.Background(), Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{"ws-1": {}}}, "ws-1", transport)
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if err := session.Subscribe(context.Background(), SubscriptionRequest{}); err != nil {
		t.Fatalf("Subscribe() error = %v", err)
	}
	hub.Publish(Event{EventID: "alarm-1", Type: "alarm.created", WorkspaceID: "ws-1", Data: []byte(`{}`)})
	select {
	case <-session.Done():
	case <-time.After(time.Second):
		t.Fatal("write failure did not close session")
	}
	done := make(chan struct{})
	go func() {
		_ = session.Close()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Close() deadlocked after writer failure")
	}
}

type recordingTransport struct {
	mu       sync.Mutex
	events   []Event
	block    chan struct{}
	writeErr error
	closed   bool
}

type recordingCapacityObserver struct {
	mu     sync.Mutex
	events []string
}

func (o *recordingCapacityObserver) RecordCapacityEvent(component, outcome string) {
	o.mu.Lock()
	o.events = append(o.events, component+":"+outcome)
	o.mu.Unlock()
}

func (o *recordingCapacityObserver) has(component, outcome string) bool {
	want := component + ":" + outcome
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, event := range o.events {
		if event == want {
			return true
		}
	}
	return false
}

func newRecordingTransport() *recordingTransport {
	return &recordingTransport{}
}

func (t *recordingTransport) Write(_ context.Context, event Event) error {
	if t.block != nil {
		<-t.block
	}
	if t.writeErr != nil {
		return t.writeErr
	}
	t.mu.Lock()
	t.events = append(t.events, event)
	t.mu.Unlock()
	return nil
}

func (t *recordingTransport) Close() error {
	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()
	return nil
}

func waitForEvent(t *testing.T, transport *recordingTransport, eventType string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		transport.mu.Lock()
		for _, event := range transport.events {
			if event.Type == eventType {
				transport.mu.Unlock()
				return
			}
		}
		transport.mu.Unlock()
		runtime.Gosched()
	}
	t.Fatalf("event %q was not written", eventType)
}

func countEvents(transport *recordingTransport, eventType string) int {
	transport.mu.Lock()
	defer transport.mu.Unlock()
	count := 0
	for _, event := range transport.events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

type fakeRecovery struct {
	snapshot      []Event
	replay        []Event
	snapshotCalls int
	replayCalls   int
}

func (r *fakeRecovery) Snapshot(context.Context, string, SubscriptionRequest) ([]Event, error) {
	r.snapshotCalls++
	return r.snapshot, nil
}

func (r *fakeRecovery) Replay(context.Context, string, string, SubscriptionRequest) ([]Event, error) {
	r.replayCalls++
	return r.replay, nil
}
