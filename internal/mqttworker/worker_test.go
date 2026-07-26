package mqttworker

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestParseBuildsStableDedupKeyAndCopiesPayload(t *testing.T) {
	raw := RawMessage{Topic: "thing/product/GW-1/services_reply", Payload: []byte(`{"gateway":"GW-1","tid":"tid-1","bid":"bid-1","method":"sim_status_refresh","data":{}}`)}
	message, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	raw.Payload[0] = 'X'
	if message.Raw.Payload[0] == 'X' || message.DedupKey != "GW-1|tid-1|bid-1|sim_status_refresh|DEVICE_TO_CLOUD" {
		t.Fatalf("parsed message was not isolated: %+v", message)
	}
}

func TestWorkerDeduplicatesAndQuarantinesMalformedMessages(t *testing.T) {
	handler := &recordingHandler{}
	quarantine := &MemoryQuarantine{}
	worker, err := New(Config{ShardCount: 2, QueueSize: 4}, handler, NewMemoryDeduplicator(), quarantine)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	ctx := context.Background()
	if err := worker.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer worker.Close()
	raw := RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"tid":"t1","bid":"b1","gateway":"GW-1","method":"sim/event","data":{}}`)}
	if err := worker.Enqueue(ctx, raw); err != nil {
		t.Fatalf("first enqueue error = %v", err)
	}
	if err := worker.Enqueue(ctx, raw); err != nil {
		t.Fatalf("duplicate enqueue error = %v", err)
	}
	if err := worker.Enqueue(ctx, RawMessage{Topic: "bad/topic", Payload: []byte(`{`)}); err != nil {
		t.Fatalf("malformed enqueue error = %v", err)
	}
	waitFor(t, func() bool {
		return worker.Stats().Handled == 1 && worker.Stats().Duplicates == 1 && worker.Stats().Quarantined == 1
	})
}

func TestWorkerHasBoundedQueueAndRetriesTransientErrors(t *testing.T) {
	handler := &recordingHandler{transientFailures: 1}
	quarantine := &MemoryQuarantine{}
	worker, err := New(Config{
		ShardCount: 1, QueueSize: 1,
		Retry: RetryPolicy{MaxAttempts: 2},
	}, handler, NewMemoryDeduplicator(), quarantine)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer worker.Close()
	message := RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"t1","bid":"b1","method":"sim/event","data":{}}`)}
	if err := worker.Enqueue(context.Background(), message); err != nil {
		t.Fatalf("enqueue error = %v", err)
	}
	waitFor(t, func() bool { return worker.Stats().Handled == 1 && handler.attempts() == 2 })

	blockedHandler := &recordingHandler{blockFirst: make(chan struct{})}
	blockedWorker, err := New(Config{ShardCount: 1, QueueSize: 1}, blockedHandler, NewMemoryDeduplicator(), &MemoryQuarantine{})
	if err != nil {
		t.Fatalf("New(blocked) error = %v", err)
	}
	if err := blockedWorker.Start(context.Background()); err != nil {
		t.Fatalf("Start(blocked) error = %v", err)
	}
	defer blockedWorker.Close()
	if err := blockedWorker.Enqueue(context.Background(), message); err != nil {
		t.Fatalf("blocked enqueue error = %v", err)
	}
	waitFor(t, func() bool { return blockedHandler.attempts() == 1 })
	second := RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"t2","bid":"b2","method":"sim/event","data":{}}`)}
	if err := blockedWorker.Enqueue(context.Background(), second); err != nil {
		t.Fatalf("queue fill enqueue error = %v", err)
	}
	third := RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"t3","bid":"b3","method":"sim/event","data":{}}`)}
	if err := blockedWorker.Enqueue(context.Background(), third); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected queue full, got %v", err)
	}
	close(blockedHandler.blockFirst)
}

func TestWorkerCloseRejectsNewMessages(t *testing.T) {
	worker, err := New(Config{ShardCount: 1, QueueSize: 1}, &recordingHandler{}, NewMemoryDeduplicator(), &MemoryQuarantine{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if err := worker.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	err = worker.Enqueue(context.Background(), RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"data":{}}`)})
	if !errors.Is(err, ErrClosed) {
		t.Fatalf("enqueue after close error = %v", err)
	}
}

func TestWorkerQuarantinesPermanentHandlerFailures(t *testing.T) {
	handler := &recordingHandler{permanentFailure: true}
	quarantine := &MemoryQuarantine{}
	worker, err := New(Config{ShardCount: 1, QueueSize: 2}, handler, NewMemoryDeduplicator(), quarantine)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer worker.Close()
	if err := worker.Enqueue(context.Background(), RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"t1","bid":"b1","method":"sim/event","data":{}}`)}); err != nil {
		t.Fatalf("enqueue error = %v", err)
	}
	waitFor(t, func() bool { return worker.Stats().Quarantined == 1 })
	if len(quarantine.Messages()) != 1 || !errors.Is(quarantine.Messages()[0].Reason, ErrPermanent) {
		t.Fatalf("permanent handler failure was not quarantined: %+v", quarantine.Messages())
	}
}

type recordingHandler struct {
	mu                sync.Mutex
	handled           int
	transientFailures int
	blockFirst        chan struct{}
	permanentFailure  bool
}

func (h *recordingHandler) Handle(_ context.Context, _ Message) error {
	h.mu.Lock()
	h.handled++
	attempt := h.handled
	h.mu.Unlock()
	if attempt == 1 && h.blockFirst != nil {
		<-h.blockFirst
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.permanentFailure {
		return ErrPermanent
	}
	if h.transientFailures > 0 {
		h.transientFailures--
		return ErrTransient
	}
	return nil
}

func (h *recordingHandler) attempts() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.handled
}

func waitFor(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		runtime.Gosched()
	}
	t.Fatal("condition was not met before timeout")
}
