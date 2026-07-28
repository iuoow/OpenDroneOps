package capacitycheck

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/mqttworker"
	"github.com/iuoow/OpenDroneOps/internal/realtime"
	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

type Config struct {
	Sessions      int
	Events        int
	Timeout       time.Duration
	MaxP95Latency time.Duration
}

type Report struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Passed      bool             `json:"passed"`
	Scenarios   []ScenarioResult `json:"scenarios"`
}

type ScenarioResult struct {
	Name          string         `json:"name"`
	Passed        bool           `json:"passed"`
	Duration      string         `json:"duration"`
	P95Latency    string         `json:"p95_latency,omitempty"`
	Checks        map[string]any `json:"checks"`
	FailureReason string         `json:"failure_reason,omitempty"`
}

func DefaultConfig() Config {
	return Config{Sessions: 8, Events: 64, Timeout: 5 * time.Second, MaxP95Latency: 500 * time.Millisecond}
}

func Run(ctx context.Context, config Config) (Report, error) {
	if ctx == nil {
		return Report{}, errors.New("capacity check context is required")
	}
	if config.Sessions < 1 || config.Events < 1 || config.Timeout <= 0 || config.MaxP95Latency <= 0 {
		return Report{}, errors.New("capacity check sessions, events, timeout, and max p95 latency must be positive")
	}
	report := Report{GeneratedAt: time.Now().UTC()}
	for _, scenario := range []func(context.Context, Config) ScenarioResult{
		websocketFanout, websocketSlowClient, mqttHotKeyRecovery, realtimeRelayDedupe,
	} {
		result := scenario(ctx, config)
		report.Scenarios = append(report.Scenarios, result)
		if !result.Passed {
			report.Passed = false
		}
	}
	if len(report.Scenarios) > 0 && !containsFailure(report.Scenarios) {
		report.Passed = true
	}
	return report, nil
}

func containsFailure(results []ScenarioResult) bool {
	for _, result := range results {
		if !result.Passed {
			return true
		}
	}
	return false
}

func websocketFanout(ctx context.Context, config Config) ScenarioResult {
	started := time.Now()
	result := ScenarioResult{Name: "websocket_fanout", Checks: map[string]any{"sessions": config.Sessions, "events": config.Events}}
	hub, err := websockethub.New(websockethub.Config{
		QueueSize: config.Events + 2, MaxSessionsPerWorkspace: config.Sessions, EventDedupeCapacity: config.Events + 2,
	})
	if err != nil {
		return failed(result, started, err)
	}
	defer hub.Close()
	transports := make([]*latencyTransport, 0, config.Sessions)
	for index := 0; index < config.Sessions; index++ {
		transport := &latencyTransport{}
		session, connectErr := hub.Connect(ctx, principal("workspace-load"), "workspace-load", transport)
		if connectErr != nil {
			return failed(result, started, connectErr)
		}
		defer session.Close()
		if subscribeErr := session.Subscribe(ctx, websockethub.SubscriptionRequest{Channels: []string{"alarm"}}); subscribeErr != nil {
			return failed(result, started, subscribeErr)
		}
		transports = append(transports, transport)
	}
	for index := 0; index < config.Events; index++ {
		hub.Publish(websockethub.Event{
			EventID: fmt.Sprintf("load-alarm-%d", index), Type: "alarm.created", SchemaVersion: "1.0",
			WorkspaceID: "workspace-load", AggregateID: "device-1", OccurredAt: time.Now().UTC(), Data: []byte(`{}`),
		})
	}
	expected := config.Sessions * config.Events
	if !wait(ctx, config.Timeout, func() bool {
		count := 0
		for _, transport := range transports {
			count += transport.count("alarm.created")
		}
		return count == expected
	}) {
		result.Checks["received"] = totalEvents(transports, "alarm.created")
		result.Checks["expected"] = expected
		return failed(result, started, errors.New("websocket fanout did not complete before timeout"))
	}
	latencies := make([]time.Duration, 0, expected)
	for _, transport := range transports {
		latencies = append(latencies, transport.latencies()...)
	}
	p95 := percentile(latencies, 0.95)
	result.P95Latency = p95.String()
	result.Checks["received"] = len(latencies)
	result.Checks["expected"] = expected
	result.Checks["slow_client_disconnects"] = hub.Stats().SlowClientDisconnects
	if p95 > config.MaxP95Latency || hub.Stats().SlowClientDisconnects != 0 {
		return failed(result, started, fmt.Errorf("websocket fanout p95=%s, slow_disconnects=%d", p95, hub.Stats().SlowClientDisconnects))
	}
	result.Passed = true
	result.Duration = time.Since(started).String()
	return result
}

func websocketSlowClient(ctx context.Context, config Config) ScenarioResult {
	started := time.Now()
	result := ScenarioResult{Name: "websocket_slow_client", Checks: map[string]any{}}
	hub, err := websockethub.New(websockethub.Config{QueueSize: 1})
	if err != nil {
		return failed(result, started, err)
	}
	transport := newBlockingTransport()
	session, err := hub.Connect(ctx, principal("workspace-load"), "workspace-load", transport)
	if err != nil {
		hub.Close()
		return failed(result, started, err)
	}
	if err := session.Subscribe(ctx, websockethub.SubscriptionRequest{}); err != nil {
		close(transport.release)
		hub.Close()
		return failed(result, started, err)
	}
	if !wait(ctx, config.Timeout, func() bool { return transport.started.Load() }) {
		close(transport.release)
		hub.Close()
		return failed(result, started, errors.New("slow client writer did not start"))
	}
	hub.Publish(websockethub.Event{EventID: "slow-telemetry", Type: "device.telemetry", WorkspaceID: "workspace-load", AggregateID: "device-1", Data: []byte(`{}`)})
	hub.Publish(websockethub.Event{EventID: "slow-alarm", Type: "alarm.created", WorkspaceID: "workspace-load", AggregateID: "device-1", Data: []byte(`{}`)})
	if !wait(ctx, config.Timeout, func() bool {
		select {
		case <-session.Done():
			return true
		default:
			return false
		}
	}) {
		close(transport.release)
		hub.Close()
		return failed(result, started, errors.New("slow client was not disconnected"))
	}
	result.Checks["slow_client_disconnects"] = hub.Stats().SlowClientDisconnects
	close(transport.release)
	_ = session.Close()
	hub.Close()
	if hub.Stats().SlowClientDisconnects != 1 {
		return failed(result, started, fmt.Errorf("slow client disconnects=%d", hub.Stats().SlowClientDisconnects))
	}
	result.Passed = true
	result.Duration = time.Since(started).String()
	return result
}

func mqttHotKeyRecovery(ctx context.Context, config Config) ScenarioResult {
	started := time.Now()
	result := ScenarioResult{Name: "mqtt_hot_key_recovery", Checks: map[string]any{}}
	handler := newBlockingHandler()
	worker, err := mqttworker.New(mqttworker.Config{ShardCount: 1, QueueSize: 4, MaxPendingPerKey: 1}, handler, mqttworker.NewMemoryDeduplicator(), &mqttworker.MemoryQuarantine{})
	if err != nil {
		return failed(result, started, err)
	}
	if err := worker.Start(ctx); err != nil {
		return failed(result, started, err)
	}
	defer func() { handler.unblock(); _ = worker.Close() }()
	first, second, third := rawMessage("h-1"), rawMessage("h-2"), rawMessage("h-3")
	if err := worker.Enqueue(ctx, first); err != nil {
		return failed(result, started, err)
	}
	if !wait(ctx, config.Timeout, handler.hasStarted) {
		return failed(result, started, errors.New("mqtt handler did not start"))
	}
	if err := worker.Enqueue(ctx, second); err != nil {
		return failed(result, started, err)
	}
	if err := worker.Enqueue(ctx, third); !errors.Is(err, mqttworker.ErrHotKeyBackpressure) {
		return failed(result, started, fmt.Errorf("hot key rejection=%v", err))
	}
	handler.unblock()
	if !wait(ctx, config.Timeout, func() bool { return handler.count() == 2 }) {
		return failed(result, started, errors.New("accepted MQTT messages did not recover"))
	}
	if err := worker.Enqueue(ctx, third); err != nil {
		return failed(result, started, err)
	}
	if !wait(ctx, config.Timeout, func() bool { return handler.count() == 3 }) {
		return failed(result, started, errors.New("retried MQTT message was not handled"))
	}
	stats := worker.Stats()
	result.Checks["handled"] = stats.Handled
	result.Checks["hot_key_backpressure"] = stats.HotKeyBackpressure
	if stats.Handled != 3 || stats.HotKeyBackpressure != 1 {
		return failed(result, started, fmt.Errorf("mqtt stats=%+v", stats))
	}
	result.Passed = true
	result.Duration = time.Since(started).String()
	return result
}

func realtimeRelayDedupe(ctx context.Context, config Config) ScenarioResult {
	started := time.Now()
	result := ScenarioResult{Name: "realtime_relay_dedupe", Checks: map[string]any{}}
	bus := newMemoryBus()
	hubA, err := websockethub.New(websockethub.Config{QueueSize: 4})
	if err != nil {
		return failed(result, started, err)
	}
	hubB, err := websockethub.New(websockethub.Config{QueueSize: 4})
	if err != nil {
		hubA.Close()
		return failed(result, started, err)
	}
	defer hubA.Close()
	defer hubB.Close()
	relayA, err := realtime.NewRelay(realtime.Config{NodeID: "capacity-a"}, hubA, bus)
	if err != nil {
		return failed(result, started, err)
	}
	relayB, err := realtime.NewRelay(realtime.Config{NodeID: "capacity-b"}, hubB, bus)
	if err != nil {
		return failed(result, started, err)
	}
	if err := relayA.Start(ctx); err != nil {
		return failed(result, started, err)
	}
	if err := relayB.Start(ctx); err != nil {
		_ = relayA.Close()
		return failed(result, started, err)
	}
	defer relayA.Close()
	defer relayB.Close()
	transport := &latencyTransport{}
	session, err := hubB.Connect(ctx, principal("workspace-load"), "workspace-load", transport)
	if err != nil {
		return failed(result, started, err)
	}
	defer session.Close()
	if err := session.Subscribe(ctx, websockethub.SubscriptionRequest{Channels: []string{"alarm"}}); err != nil {
		return failed(result, started, err)
	}
	event := websockethub.Event{EventID: "relay-event", Type: "alarm.created", WorkspaceID: "workspace-load", Data: []byte(`{}`)}
	if err := relayA.PublishContext(ctx, event); err != nil {
		return failed(result, started, err)
	}
	if err := relayA.PublishContext(ctx, event); err != nil {
		return failed(result, started, err)
	}
	if !wait(ctx, config.Timeout, func() bool { return transport.count("alarm.created") == 1 && relayB.Stats().Duplicates == 1 }) {
		return failed(result, started, errors.New("realtime duplicate was not suppressed"))
	}
	result.Checks["received"] = relayB.Stats().Received
	result.Checks["duplicates"] = relayB.Stats().Duplicates
	result.Passed = true
	result.Duration = time.Since(started).String()
	return result
}

func failed(result ScenarioResult, started time.Time, err error) ScenarioResult {
	result.Duration = time.Since(started).String()
	result.Passed = false
	result.FailureReason = err.Error()
	return result
}

func wait(ctx context.Context, timeout time.Duration, condition func() bool) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		if condition() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-deadline.C:
			return false
		case <-ticker.C:
		}
	}
}

func percentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]time.Duration(nil), values...)
	sort.Slice(sorted, func(left, right int) bool { return sorted[left] < sorted[right] })
	index := int(float64(len(sorted)-1) * percentile)
	return sorted[index]
}

func principal(workspaceID string) websockethub.Principal {
	return websockethub.Principal{Subject: "capacity-check", WorkspaceIDs: map[string]struct{}{workspaceID: {}}}
}

type latencyTransport struct {
	mu             sync.Mutex
	events         []websockethub.Event
	latencySamples []time.Duration
}

func (t *latencyTransport) Write(_ context.Context, event websockethub.Event) error {
	t.mu.Lock()
	t.events = append(t.events, event)
	if event.Type == "alarm.created" && !event.OccurredAt.IsZero() {
		t.latencySamples = append(t.latencySamples, time.Since(event.OccurredAt))
	}
	t.mu.Unlock()
	return nil
}

func (t *latencyTransport) Close() error { return nil }

func (t *latencyTransport) count(eventType string) int {
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

func (t *latencyTransport) latencies() []time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]time.Duration(nil), t.latencySamples...)
}

func totalEvents(transports []*latencyTransport, eventType string) int {
	total := 0
	for _, transport := range transports {
		total += transport.count(eventType)
	}
	return total
}

type blockingTransport struct {
	started atomic.Bool
	release chan struct{}
}

func newBlockingTransport() *blockingTransport {
	return &blockingTransport{release: make(chan struct{})}
}

func (t *blockingTransport) Write(_ context.Context, _ websockethub.Event) error {
	t.started.Store(true)
	<-t.release
	return nil
}

func (t *blockingTransport) Close() error { return nil }

type blockingHandler struct {
	started atomic.Bool
	release chan struct{}
	once    sync.Once
	counted atomic.Uint64
}

func newBlockingHandler() *blockingHandler { return &blockingHandler{release: make(chan struct{})} }

func (h *blockingHandler) Handle(_ context.Context, _ mqttworker.Message) error {
	count := h.counted.Add(1)
	if count == 1 {
		h.started.Store(true)
		<-h.release
	}
	return nil
}

func (h *blockingHandler) hasStarted() bool { return h.started.Load() }
func (h *blockingHandler) count() uint64    { return h.counted.Load() }
func (h *blockingHandler) unblock()         { h.once.Do(func() { close(h.release) }) }

func rawMessage(tid string) mqttworker.RawMessage {
	return mqttworker.RawMessage{Topic: "thing/product/GW-1/events", Payload: []byte(`{"gateway":"GW-1","tid":"` + tid + `","bid":"b-` + tid + `","method":"sim/event","data":{}}`)}
}
