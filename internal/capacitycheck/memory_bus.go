package capacitycheck

import (
	"context"
	"sync"

	"github.com/iuoow/OpenDroneOps/internal/realtime"
)

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

func (b *memoryBus) Subscribe(_ context.Context, channel string) (realtime.Subscription, error) {
	subscription := &memorySubscription{bus: b, channel: channel, messages: make(chan []byte, 16)}
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
