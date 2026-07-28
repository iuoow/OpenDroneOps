package mqttworker

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/fnv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/protocol/dji"
)

var (
	ErrClosed             = errors.New("MQTT worker is closed")
	ErrQueueFull          = errors.New("MQTT worker shard queue is full")
	ErrHotKeyBackpressure = errors.New("MQTT worker key queue is full")
	ErrTransient          = errors.New("transient MQTT message handling failure")
	ErrPermanent          = errors.New("permanent MQTT message handling failure")
)

type RawMessage struct {
	Topic      string
	Payload    []byte
	QoS        byte
	Retain     bool
	ReceivedAt time.Time
}

type Message struct {
	Raw      RawMessage
	Parsed   dji.Message
	DedupKey string
	Attempts int
}

type Handler interface {
	Handle(context.Context, Message) error
}

type Deduplicator interface {
	CheckAndMark(context.Context, string) (bool, error)
}

type QuarantineSink interface {
	Quarantine(context.Context, RawMessage, error) error
}

type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{MaxAttempts: 3, InitialBackoff: 10 * time.Millisecond, MaxBackoff: time.Second}
}

type Config struct {
	ShardCount       int
	QueueSize        int
	MaxPendingPerKey int
	Retry            RetryPolicy
	OnError          func(error)
	CapacityObserver CapacityObserver
}

// CapacityObserver receives low-cardinality overload outcomes without coupling
// the worker to a concrete metrics package.
type CapacityObserver interface {
	RecordCapacityEvent(component, outcome string)
}

type Stats struct {
	Accepted           uint64
	Duplicates         uint64
	Handled            uint64
	Quarantined        uint64
	QueueFull          uint64
	HotKeyBackpressure uint64
	HandlerFailures    uint64
}

type Worker struct {
	config           Config
	handler          Handler
	deduplicator     Deduplicator
	quarantine       QuarantineSink
	queues           []*fairQueue
	onError          func(error)
	capacityObserver CapacityObserver

	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	started bool
	closed  bool
	wg      sync.WaitGroup

	accepted           atomic.Uint64
	duplicates         atomic.Uint64
	handled            atomic.Uint64
	quarantined        atomic.Uint64
	queueFull          atomic.Uint64
	hotKeyBackpressure atomic.Uint64
	handlerFailures    atomic.Uint64
}

func New(config Config, handler Handler, deduplicator Deduplicator, quarantine QuarantineSink) (*Worker, error) {
	if config.ShardCount < 1 || config.QueueSize < 1 {
		return nil, errors.New("MQTT shard count and queue size must be positive")
	}
	if config.MaxPendingPerKey == 0 || config.MaxPendingPerKey > config.QueueSize {
		config.MaxPendingPerKey = min(config.QueueSize, 64)
	}
	if config.MaxPendingPerKey < 1 {
		return nil, errors.New("MQTT maximum pending messages per key must be positive")
	}
	if handler == nil || deduplicator == nil || quarantine == nil {
		return nil, errors.New("MQTT worker handler, deduplicator, and quarantine sink are required")
	}
	if config.Retry.MaxAttempts < 1 {
		config.Retry = DefaultRetryPolicy()
	}
	if config.Retry.MaxBackoff <= 0 {
		config.Retry.MaxBackoff = time.Second
	}
	return &Worker{
		config: config, handler: handler, deduplicator: deduplicator, quarantine: quarantine,
		onError: config.OnError, capacityObserver: config.CapacityObserver,
		queues: make([]*fairQueue, config.ShardCount),
	}, nil
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.started {
		return errors.New("MQTT worker already started")
	}
	if ctx == nil {
		return errors.New("MQTT worker context is required")
	}
	w.ctx, w.cancel = context.WithCancel(ctx)
	w.started = true
	for index := range w.queues {
		w.queues[index] = newFairQueue(w.config.QueueSize, w.config.MaxPendingPerKey)
		w.wg.Add(1)
		go w.runShard(w.queues[index])
	}
	return nil
}

func (w *Worker) Enqueue(ctx context.Context, raw RawMessage) error {
	if raw.ReceivedAt.IsZero() {
		raw.ReceivedAt = time.Now().UTC()
	}
	message, err := Parse(raw)
	if err != nil {
		w.quarantineMessage(ctx, raw, err)
		return nil
	}
	w.mu.Lock()
	if !w.started || w.closed {
		w.mu.Unlock()
		return ErrClosed
	}
	workerCtx := w.ctx
	queue := w.queues[shardFor(message.DedupKey, len(w.queues))]
	w.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-workerCtx.Done():
		return ErrClosed
	default:
	}
	if err := queue.Enqueue(fairnessKey(message), message); err != nil {
		if errors.Is(err, ErrQueueFull) {
			w.queueFull.Add(1)
			w.recordCapacity("shard_queue_limit")
		}
		if errors.Is(err, ErrHotKeyBackpressure) {
			w.hotKeyBackpressure.Add(1)
			w.recordCapacity("hot_key_limit")
		}
		return err
	}
	w.accepted.Add(1)
	return nil
}

func (w *Worker) Close() error {
	w.mu.Lock()
	if !w.started || w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.cancel()
	for _, queue := range w.queues {
		queue.Close()
	}
	w.mu.Unlock()
	w.wg.Wait()
	return nil
}

func (w *Worker) Stats() Stats {
	return Stats{
		Accepted: w.accepted.Load(), Duplicates: w.duplicates.Load(),
		Handled: w.handled.Load(), Quarantined: w.quarantined.Load(),
		QueueFull: w.queueFull.Load(), HotKeyBackpressure: w.hotKeyBackpressure.Load(),
		HandlerFailures: w.handlerFailures.Load(),
	}
}

func Parse(raw RawMessage) (Message, error) {
	parsed, err := dji.ParseMessage(raw.Topic, raw.Payload)
	if err != nil {
		return Message{}, err
	}
	return Message{
		Raw: RawMessage{
			Topic: raw.Topic, Payload: append([]byte(nil), raw.Payload...),
			QoS: raw.QoS, Retain: raw.Retain, ReceivedAt: raw.ReceivedAt,
		},
		Parsed:   parsed,
		DedupKey: dedupKey(parsed),
	}, nil
}

func dedupKey(message dji.Message) string {
	envelope := message.Envelope
	direction := string(message.Topic.Direction)
	if envelope.Gateway != "" || envelope.TID != "" || envelope.BID != "" || envelope.Method != "" {
		return strings.Join([]string{envelope.Gateway, envelope.TID, envelope.BID, envelope.Method, direction}, "|")
	}
	return message.Topic.Raw + "|" + hex.EncodeToString(message.PayloadHash[:])
}

func (w *Worker) runShard(queue *fairQueue) {
	defer w.wg.Done()
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			message, ok := queue.Dequeue(w.ctx)
			if !ok {
				return
			}
			w.process(message)
		}
	}
}

func (w *Worker) process(message Message) {
	duplicate, err := w.deduplicator.CheckAndMark(w.ctx, message.DedupKey)
	if err != nil {
		if w.onError != nil {
			w.onError(fmt.Errorf("check duplicate %q: %w", message.DedupKey, err))
		}
		return
	}
	if duplicate {
		w.duplicates.Add(1)
		return
	}
	for attempt := 1; attempt <= w.config.Retry.MaxAttempts; attempt++ {
		message.Attempts = attempt
		err := w.handler.Handle(w.ctx, message)
		if err == nil {
			w.handled.Add(1)
			return
		}
		w.handlerFailures.Add(1)
		if !errors.Is(err, ErrTransient) || attempt == w.config.Retry.MaxAttempts {
			w.quarantineMessage(w.ctx, message.Raw, err)
			return
		}
		if err := waitBackoff(w.ctx, backoff(w.config.Retry, attempt)); err != nil {
			w.quarantineMessage(context.Background(), message.Raw, err)
			return
		}
	}
}

func (w *Worker) recordCapacity(outcome string) {
	if w.capacityObserver != nil {
		w.capacityObserver.RecordCapacityEvent("mqtt_ingestion", outcome)
	}
}

func fairnessKey(message Message) string {
	if message.Parsed.Topic.DeviceSN != "" {
		return "device:" + message.Parsed.Topic.DeviceSN
	}
	if message.Parsed.Topic.GatewaySN != "" {
		return "gateway:" + message.Parsed.Topic.GatewaySN
	}
	return "topic:" + message.Parsed.Topic.Raw
}

func (w *Worker) quarantineMessage(ctx context.Context, raw RawMessage, reason error) {
	if err := w.quarantine.Quarantine(ctx, raw, reason); err != nil {
		if w.onError != nil {
			w.onError(fmt.Errorf("quarantine MQTT message: %w", err))
		}
		return
	}
	w.quarantined.Add(1)
}

func backoff(policy RetryPolicy, attempt int) time.Duration {
	delay := policy.InitialBackoff
	for index := 1; index < attempt; index++ {
		if delay >= policy.MaxBackoff/2 {
			return policy.MaxBackoff
		}
		delay *= 2
	}
	if delay > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return delay
}

func waitBackoff(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func shardFor(key string, count int) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(key))
	return int(hash.Sum32() % uint32(count))
}

type MemoryDeduplicator struct {
	mu   sync.Mutex
	keys map[string]struct{}
}

func NewMemoryDeduplicator() *MemoryDeduplicator {
	return &MemoryDeduplicator{keys: make(map[string]struct{})}
}

func (d *MemoryDeduplicator) CheckAndMark(_ context.Context, key string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.keys[key]; exists {
		return true, nil
	}
	d.keys[key] = struct{}{}
	return false, nil
}

type QuarantinedMessage struct {
	Message RawMessage
	Reason  error
}

type MemoryQuarantine struct {
	mu       sync.Mutex
	messages []QuarantinedMessage
}

func (q *MemoryQuarantine) Quarantine(_ context.Context, message RawMessage, reason error) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.messages = append(q.messages, QuarantinedMessage{
		Message: RawMessage{Topic: message.Topic, Payload: append([]byte(nil), message.Payload...), QoS: message.QoS, Retain: message.Retain, ReceivedAt: message.ReceivedAt},
		Reason:  reason,
	})
	return nil
}

func (q *MemoryQuarantine) Messages() []QuarantinedMessage {
	q.mu.Lock()
	defer q.mu.Unlock()
	result := make([]QuarantinedMessage, len(q.messages))
	copy(result, q.messages)
	return result
}
