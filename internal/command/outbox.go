package command

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type Broker interface {
	Publish(context.Context, string, []byte, byte) error
}

type OutboxConfig struct {
	WorkerID         string
	BatchSize        int
	MaxAttempts      int
	LeaseDuration    time.Duration
	PublishTimeout   time.Duration
	PollInterval     time.Duration
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	Jitter           func(time.Duration) time.Duration
	OnCommandChanged func(domain.Command, time.Time)
	OnError          func(error)
}

type OutboxPublisher struct {
	repository Repository
	broker     Broker
	config     OutboxConfig
	now        func() time.Time
}

func NewOutboxPublisher(repository Repository, broker Broker, config OutboxConfig) (*OutboxPublisher, error) {
	if repository == nil || broker == nil {
		return nil, errors.New("outbox repository and broker are required")
	}
	if config.WorkerID == "" {
		return nil, errors.New("outbox worker id is required")
	}
	if config.BatchSize < 1 {
		config.BatchSize = 32
	}
	if config.MaxAttempts < 1 {
		config.MaxAttempts = 5
	}
	if config.LeaseDuration <= 0 {
		config.LeaseDuration = 30 * time.Second
	}
	if config.PublishTimeout <= 0 {
		config.PublishTimeout = 5 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = time.Second
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = time.Second
	}
	if config.MaxBackoff < config.InitialBackoff {
		config.MaxBackoff = time.Minute
	}
	if config.Jitter == nil {
		config.Jitter = secureJitter
	}
	return &OutboxPublisher{repository: repository, broker: broker, config: config, now: func() time.Time {
		return time.Now().UTC()
	}}, nil
}

func (p *OutboxPublisher) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()
	for {
		if _, err := p.RunOnce(ctx); err != nil && p.config.OnError != nil {
			p.config.OnError(err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *OutboxPublisher) RunOnce(ctx context.Context) (int, error) {
	now := p.now()
	deliveries, err := p.repository.LeaseOutbox(ctx, p.config.WorkerID, p.config.BatchSize, now, p.config.LeaseDuration)
	if err != nil {
		return 0, fmt.Errorf("lease outbox: %w", err)
	}
	var failures []error
	for _, delivery := range deliveries {
		event := delivery.Event
		publishCtx, cancel := context.WithTimeout(ctx, p.config.PublishTimeout)
		err := p.broker.Publish(publishCtx, event.Destination, event.Payload, delivery.QoS)
		cancel()
		if err == nil {
			transitionAt := p.now()
			command, changed, completeErr := p.repository.MarkPublished(ctx, p.config.WorkerID, event.ID, event.AggregateID, transitionAt)
			if completeErr != nil {
				failures = append(failures, fmt.Errorf("complete outbox %s: %w", event.ID, completeErr))
			} else if changed && p.config.OnCommandChanged != nil {
				p.config.OnCommandChanged(command, transitionAt)
			}
			continue
		}
		maxAttempts := p.config.MaxAttempts
		if delivery.RiskLevel == domain.RiskHigh {
			maxAttempts = 1
		}
		if event.AttemptCount >= maxAttempts {
			transitionAt := p.now()
			command, changed, failErr := p.repository.MarkFailed(ctx, p.config.WorkerID, event.ID, event.AggregateID, transitionAt, err.Error())
			if failErr != nil {
				failures = append(failures, fmt.Errorf("fail outbox %s: %w", event.ID, failErr))
			} else if changed && p.config.OnCommandChanged != nil {
				p.config.OnCommandChanged(command, transitionAt)
			}
			continue
		}
		nextAttempt := p.now().Add(p.backoff(event.AttemptCount))
		if retryErr := p.repository.MarkRetry(ctx, p.config.WorkerID, event.ID, nextAttempt, err.Error()); retryErr != nil {
			failures = append(failures, fmt.Errorf("retry outbox %s: %w", event.ID, retryErr))
		}
	}
	return len(deliveries), errors.Join(failures...)
}

func (p *OutboxPublisher) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := p.config.InitialBackoff
	for i := 1; i < attempt && delay < p.config.MaxBackoff; i++ {
		if delay > p.config.MaxBackoff/2 {
			delay = p.config.MaxBackoff
			break
		}
		delay *= 2
	}
	if delay > p.config.MaxBackoff {
		delay = p.config.MaxBackoff
	}
	return delay + p.config.Jitter(delay)
}

func secureJitter(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	limit := uint64(base / 4)
	if limit == 0 {
		return 0
	}
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0
	}
	return time.Duration(binary.LittleEndian.Uint64(buffer[:]) % (limit + 1))
}
