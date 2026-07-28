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

func TestFairQueueRoundRobinsActiveKeys(t *testing.T) {
	queue := newFairQueue(4, 3)
	for _, message := range []struct {
		key string
		id  string
	}{
		{key: "gateway:hot", id: "h-1"},
		{key: "gateway:hot", id: "h-2"},
		{key: "gateway:cool", id: "c-1"},
		{key: "gateway:hot", id: "h-3"},
	} {
		if err := queue.Enqueue(message.key, Message{DedupKey: message.id}); err != nil {
			t.Fatalf("Enqueue(%q) error = %v", message.id, err)
		}
	}
	var got []string
	for range 4 {
		message, ok := queue.Dequeue(context.Background())
		if !ok {
			t.Fatal("Dequeue() returned closed queue")
		}
		got = append(got, message.DedupKey)
	}
	want := []string{"h-1", "c-1", "h-2", "h-3"}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("fair dequeue order = %v, want %v", got, want)
		}
	}
}

func TestWorkerRejectsHotKeyWithoutPoisoningDeduplication(t *testing.T) {
	handler := newBlockingOrderHandler("h-1")
	observer := &recordingCapacityObserver{}
	worker, err := New(Config{
		ShardCount: 1, QueueSize: 4, MaxPendingPerKey: 1, CapacityObserver: observer,
	}, handler, NewMemoryDeduplicator(), &MemoryQuarantine{})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := worker.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer worker.Close()
	first := workerMessage("h-1")
	second := workerMessage("h-2")
	third := workerMessage("h-3")
	if err := worker.Enqueue(context.Background(), first); err != nil {
		t.Fatalf("first enqueue error = %v", err)
	}
	select {
	case <-handler.started:
	case <-time.After(time.Second):
		t.Fatal("first message was not started")
	}
	if err := worker.Enqueue(context.Background(), second); err != nil {
		t.Fatalf("second enqueue error = %v", err)
	}
	if err := worker.Enqueue(context.Background(), third); !errors.Is(err, ErrHotKeyBackpressure) {
		t.Fatalf("third enqueue error = %v, want hot-key backpressure", err)
	}
	if stats := worker.Stats(); stats.HotKeyBackpressure != 1 || stats.QueueFull != 0 {
		t.Fatalf("Stats() = %+v, want one hot-key rejection", stats)
	}
	if !observer.has("mqtt_ingestion", "hot_key_limit") {
		t.Fatalf("capacity events = %v", observer.events)
	}
	close(handler.release)
	waitFor(t, func() bool { return handler.count() == 2 })
	if err := worker.Enqueue(context.Background(), third); err != nil {
		t.Fatalf("retry after backpressure error = %v", err)
	}
	waitFor(t, func() bool { return handler.count() == 3 })
	if got := handler.order(); got[2] != "h-3" {
		t.Fatalf("retried message was not handled: %v", got)
	}
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

type blockingOrderHandler struct {
	blockTID string
	started  chan struct{}
	release  chan struct{}
	once     sync.Once
	mu       sync.Mutex
	tids     []string
}

func newBlockingOrderHandler(blockTID string) *blockingOrderHandler {
	return &blockingOrderHandler{blockTID: blockTID, started: make(chan struct{}), release: make(chan struct{})}
}

func (h *blockingOrderHandler) Handle(_ context.Context, message Message) error {
	tid := message.Parsed.Envelope.TID
	h.mu.Lock()
	h.tids = append(h.tids, tid)
	h.mu.Unlock()
	if tid == h.blockTID {
		h.once.Do(func() { close(h.started) })
		<-h.release
	}
	return nil
}

func (h *blockingOrderHandler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.tids)
}

func (h *blockingOrderHandler) order() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.tids...)
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

func workerMessage(tid string) RawMessage {
	return RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"` + tid + `","bid":"b-` + tid + `","method":"sim/event","data":{}}`)}
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
