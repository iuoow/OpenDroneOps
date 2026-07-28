package realtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

func TestRelayDeliversAcrossInstancesOnce(t *testing.T) {
	bus := newMemoryBus()
	hubA, err := websockethub.New(websockethub.Config{QueueSize: 8})
	if err != nil {
		t.Fatalf("New hub A: %v", err)
	}
	hubB, err := websockethub.New(websockethub.Config{QueueSize: 8})
	if err != nil {
		t.Fatalf("New hub B: %v", err)
	}
	defer hubA.Close()
	defer hubB.Close()
	relayA, err := NewRelay(Config{NodeID: "node-a"}, hubA, bus)
	if err != nil {
		t.Fatalf("NewRelay A: %v", err)
	}
	relayB, err := NewRelay(Config{NodeID: "node-b"}, hubB, bus)
	if err != nil {
		t.Fatalf("NewRelay B: %v", err)
	}
	if err := relayA.Start(context.Background()); err != nil {
		t.Fatalf("Start relay A: %v", err)
	}
	if err := relayB.Start(context.Background()); err != nil {
		t.Fatalf("Start relay B: %v", err)
	}
	defer relayA.Close()
	defer relayB.Close()

	transport := &recordingTransport{}
	session, err := hubB.Connect(context.Background(), principal("workspace-1"), "workspace-1", transport)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer session.Close()
	if err := session.Subscribe(context.Background(), websockethub.SubscriptionRequest{Channels: []string{"alarm"}}); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	event := websockethub.Event{EventID: "alarm-1", Type: "alarm.created", SchemaVersion: "1.0", WorkspaceID: "workspace-1", AggregateID: "device-1", OccurredAt: time.Now().UTC(), Data: []byte(`{}`)}
	if err := relayA.PublishContext(context.Background(), event); err != nil {
		t.Fatalf("first PublishContext: %v", err)
	}
	if err := relayA.PublishContext(context.Background(), event); err != nil {
		t.Fatalf("duplicate PublishContext: %v", err)
	}
	waitFor(t, func() bool { return transport.count("alarm.created") == 1 && relayB.Stats().Duplicates == 1 })
	if stats := relayB.Stats(); stats.Received != 1 || stats.Duplicates != 1 {
		t.Fatalf("relay B stats = %+v", stats)
	}
}

func TestRelayRejectsInvalidEventAndTracksPublishFailure(t *testing.T) {
	hub, err := websockethub.New(websockethub.Config{QueueSize: 2})
	if err != nil {
		t.Fatalf("New hub: %v", err)
	}
	defer hub.Close()
	bus := &failingBus{err: errors.New("redis unavailable")}
	relay, err := NewRelay(Config{NodeID: "node-a"}, hub, bus)
	if err != nil {
		t.Fatalf("NewRelay: %v", err)
	}
	if err := relay.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer relay.Close()
	if err := relay.PublishContext(context.Background(), websockethub.Event{WorkspaceID: "workspace-1"}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("invalid PublishContext error = %v", err)
	}
	event := websockethub.Event{EventID: "event-1", WorkspaceID: "workspace-1", Type: "alarm.created", Data: []byte(`{}`)}
	if err := relay.PublishContext(context.Background(), event); err == nil || !errors.Is(err, bus.err) {
		t.Fatalf("PublishContext error = %v, want bus failure", err)
	}
	if relay.Stats().PublishFailures != 1 {
		t.Fatalf("relay stats = %+v", relay.Stats())
	}
}

func principal(workspaceID string) websockethub.Principal {
	return websockethub.Principal{Subject: "operator", WorkspaceIDs: map[string]struct{}{workspaceID: {}}}
}

type recordingTransport struct {
	mu     sync.Mutex
	events []websockethub.Event
}

func (t *recordingTransport) Write(_ context.Context, event websockethub.Event) error {
	t.mu.Lock()
	t.events = append(t.events, event)
	t.mu.Unlock()
	return nil
}

func (t *recordingTransport) Close() error { return nil }

func (t *recordingTransport) count(eventType string) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	for _, event := range t.events {
		if event.Type == eventType {
			count++
		}
	}
	return count
}

type memoryBus struct {
	mu            sync.Mutex
	subscriptions map[string]map[*memorySubscription]struct{}
}

func newMemoryBus() *memoryBus {
	return &memoryBus{subscriptions: make(map[string]map[*memorySubscription]struct{})}
}

func (b *memoryBus) Publish(_ context.Context, channel string, payload []byte) error {
	b.mu.Lock()
	subscriptions := make([]*memorySubscription, 0, len(b.subscriptions[channel]))
	for subscription := range b.subscriptions[channel] {
		subscriptions = append(subscriptions, subscription)
	}
	b.mu.Unlock()
	for _, subscription := range subscriptions {
		select {
		case subscription.messages <- append([]byte(nil), payload...):
		default:
		}
	}
	return nil
}

func (b *memoryBus) Subscribe(_ context.Context, channel string) (Subscription, error) {
	subscription := &memorySubscription{bus: b, channel: channel, messages: make(chan []byte, 8)}
	b.mu.Lock()
	if b.subscriptions[channel] == nil {
		b.subscriptions[channel] = make(map[*memorySubscription]struct{})
	}
	b.subscriptions[channel][subscription] = struct{}{}
	b.mu.Unlock()
	return subscription, nil
}

type memorySubscription struct {
	bus      *memoryBus
	channel  string
	messages chan []byte
	once     sync.Once
}

func (s *memorySubscription) Messages() <-chan []byte { return s.messages }

func (s *memorySubscription) Close() error {
	s.once.Do(func() {
		s.bus.mu.Lock()
		delete(s.bus.subscriptions[s.channel], s)
		s.bus.mu.Unlock()
		close(s.messages)
	})
	return nil
}

type failingBus struct{ err error }

func (b *failingBus) Publish(context.Context, string, []byte) error { return b.err }

func (b *failingBus) Subscribe(context.Context, string) (Subscription, error) {
	return &memorySubscription{bus: newMemoryBus(), channel: "test", messages: make(chan []byte)}, nil
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
