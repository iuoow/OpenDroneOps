package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

var (
	ErrRelayNotStarted = errors.New("realtime relay is not started")
	ErrInvalidEvent    = errors.New("realtime event requires event_id and workspace_id")
)

const protocolVersion = "1.0"

// Bus carries ephemeral cross-instance notifications. It is not an event store:
// callers must use the Hub recovery provider for missed messages.
type Bus interface {
	Publish(context.Context, string, []byte) error
	Subscribe(context.Context, string) (Subscription, error)
}

type Subscription interface {
	Messages() <-chan []byte
	Close() error
}

type CapacityObserver interface {
	RecordCapacityEvent(component, outcome string)
}

type Config struct {
	NodeID           string
	Channel          string
	DedupeCapacity   int
	PublishTimeout   time.Duration
	OnError          func(error)
	CapacityObserver CapacityObserver
}

type Stats struct {
	Published       uint64
	Received        uint64
	Duplicates      uint64
	InvalidMessages uint64
	PublishFailures uint64
}

type envelope struct {
	Version string             `json:"version"`
	Origin  string             `json:"origin"`
	Event   websockethub.Event `json:"event"`
}

type Relay struct {
	hub    *websockethub.Hub
	bus    Bus
	config Config

	mu           sync.Mutex
	ctx          context.Context
	cancel       context.CancelFunc
	subscription Subscription
	started      bool
	closed       bool
	wg           sync.WaitGroup

	dedupeMu sync.Mutex
	seen     map[string]struct{}
	order    []string

	published       atomic.Uint64
	received        atomic.Uint64
	duplicates      atomic.Uint64
	invalidMessages atomic.Uint64
	publishFailures atomic.Uint64
}

func NewRelay(config Config, hub *websockethub.Hub, bus Bus) (*Relay, error) {
	if hub == nil || bus == nil {
		return nil, errors.New("realtime hub and bus are required")
	}
	if config.NodeID == "" {
		return nil, errors.New("realtime node id is required")
	}
	if config.Channel == "" {
		config.Channel = "opendroneops:realtime:v1"
	}
	if config.DedupeCapacity == 0 {
		config.DedupeCapacity = 4096
	}
	if config.DedupeCapacity < 1 {
		return nil, errors.New("realtime dedupe capacity must be positive")
	}
	if config.PublishTimeout <= 0 {
		config.PublishTimeout = 2 * time.Second
	}
	return &Relay{
		hub: hub, bus: bus, config: config,
		seen: make(map[string]struct{}, config.DedupeCapacity),
	}, nil
}

func (r *Relay) Start(ctx context.Context) error {
	if ctx == nil {
		return errors.New("realtime relay context is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.started {
		return errors.New("realtime relay already started")
	}
	if r.closed {
		return errors.New("realtime relay is closed")
	}
	subscription, err := r.bus.Subscribe(ctx, r.config.Channel)
	if err != nil {
		return fmt.Errorf("subscribe realtime channel: %w", err)
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.subscription = subscription
	r.started = true
	r.wg.Add(1)
	go r.receiveLoop()
	return nil
}

// Publish makes an event visible to local sessions and then asks the ephemeral
// bus to deliver it to sessions connected to other instances.
func (r *Relay) Publish(event websockethub.Event) {
	if err := r.PublishContext(context.Background(), event); err != nil && r.config.OnError != nil {
		r.config.OnError(err)
	}
}

func (r *Relay) PublishContext(ctx context.Context, event websockethub.Event) error {
	if event.EventID == "" || event.WorkspaceID == "" {
		return ErrInvalidEvent
	}
	r.mu.Lock()
	started := r.started && !r.closed
	r.mu.Unlock()
	if !started {
		return ErrRelayNotStarted
	}
	r.hub.Publish(event)
	payload, err := json.Marshal(envelope{Version: protocolVersion, Origin: r.config.NodeID, Event: event})
	if err != nil {
		return fmt.Errorf("marshal realtime event: %w", err)
	}
	publishCtx, cancel := context.WithTimeout(ctx, r.config.PublishTimeout)
	defer cancel()
	if err := r.bus.Publish(publishCtx, r.config.Channel, payload); err != nil {
		r.publishFailures.Add(1)
		r.recordCapacity("publish_failure")
		return fmt.Errorf("publish realtime event: %w", err)
	}
	r.published.Add(1)
	return nil
}

func (r *Relay) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	if r.cancel != nil {
		r.cancel()
	}
	subscription := r.subscription
	r.mu.Unlock()
	if subscription != nil {
		_ = subscription.Close()
	}
	r.wg.Wait()
	return nil
}

func (r *Relay) Stats() Stats {
	return Stats{
		Published: r.published.Load(), Received: r.received.Load(),
		Duplicates: r.duplicates.Load(), InvalidMessages: r.invalidMessages.Load(),
		PublishFailures: r.publishFailures.Load(),
	}
}

func (r *Relay) receiveLoop() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case payload, ok := <-r.subscription.Messages():
			if !ok {
				return
			}
			r.receive(payload)
		}
	}
}

func (r *Relay) receive(payload []byte) {
	var message envelope
	if err := json.Unmarshal(payload, &message); err != nil || message.Version != protocolVersion || message.Origin == "" || message.Event.EventID == "" || message.Event.WorkspaceID == "" {
		r.invalidMessages.Add(1)
		r.recordCapacity("invalid_message")
		return
	}
	if message.Origin == r.config.NodeID {
		return
	}
	if r.markSeen(message.Origin + ":" + message.Event.EventID) {
		r.duplicates.Add(1)
		r.recordCapacity("duplicate_event")
		return
	}
	r.hub.Publish(message.Event)
	r.received.Add(1)
}

func (r *Relay) markSeen(key string) bool {
	r.dedupeMu.Lock()
	defer r.dedupeMu.Unlock()
	if _, exists := r.seen[key]; exists {
		return true
	}
	r.seen[key] = struct{}{}
	r.order = append(r.order, key)
	if len(r.order) > r.config.DedupeCapacity {
		oldest := r.order[0]
		delete(r.seen, oldest)
		r.order = r.order[1:]
	}
	return false
}

func (r *Relay) recordCapacity(outcome string) {
	if r.config.CapacityObserver != nil {
		r.config.CapacityObserver.RecordCapacityEvent("realtime", outcome)
	}
}
