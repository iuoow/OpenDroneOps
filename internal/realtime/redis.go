package realtime

import (
	"context"
	"errors"
	"sync"

	"github.com/redis/go-redis/v9"
)

type RedisBus struct {
	client *redis.Client
}

func NewRedisBus(client *redis.Client) (*RedisBus, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	return &RedisBus{client: client}, nil
}

func (b *RedisBus) Publish(ctx context.Context, channel string, payload []byte) error {
	return b.client.Publish(ctx, channel, payload).Err()
}

func (b *RedisBus) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	pubsub := b.client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, err
	}
	subscription := &redisSubscription{
		pubsub: pubsub, messages: make(chan []byte, 64), done: make(chan struct{}),
	}
	go subscription.forward(ctx)
	return subscription, nil
}

type redisSubscription struct {
	pubsub   *redis.PubSub
	messages chan []byte
	done     chan struct{}
	once     sync.Once
}

func (s *redisSubscription) Messages() <-chan []byte {
	return s.messages
}

func (s *redisSubscription) Close() error {
	var err error
	s.once.Do(func() {
		close(s.done)
		err = s.pubsub.Close()
	})
	return err
}

func (s *redisSubscription) forward(ctx context.Context) {
	defer close(s.messages)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case message, ok := <-s.pubsub.Channel():
			if !ok {
				return
			}
			payload := append([]byte(nil), message.Payload...)
			select {
			case s.messages <- payload:
			case <-ctx.Done():
				return
			case <-s.done:
				return
			}
		}
	}
}
