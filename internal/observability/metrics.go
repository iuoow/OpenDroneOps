package observability

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type requestMetric struct {
	count uint64
	sum   float64
}

type CapacityEvent struct {
	Component      string `json:"component"`
	Outcome        string `json:"outcome"`
	Count          uint64 `json:"count"`
	Severity       string `json:"severity"`
	Recommendation string `json:"recommendation"`
}

type CapacitySummary struct {
	GeneratedAt      time.Time       `json:"generated_at"`
	ProcessStartedAt time.Time       `json:"process_started_at"`
	Health           string          `json:"health"`
	Events           []CapacityEvent `json:"events"`
}

// Registry is a deliberately small Prometheus text-format registry. It keeps
// the HTTP boundary observable without introducing an exporter dependency.
type Registry struct {
	mu             sync.RWMutex
	startedAt      time.Time
	requests       map[string]requestMetric
	capacityEvents map[string]uint64
}

func NewRegistry(now time.Time) *Registry {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Registry{
		startedAt:      now,
		requests:       make(map[string]requestMetric),
		capacityEvents: make(map[string]uint64),
	}
}

func (r *Registry) RecordHTTP(method, route string, status int, duration time.Duration) {
	if r == nil {
		return
	}
	key := method + "\x00" + route + "\x00" + strconv.Itoa(status)
	r.mu.Lock()
	metric := r.requests[key]
	metric.count++
	metric.sum += duration.Seconds()
	r.requests[key] = metric
	r.mu.Unlock()
}

// RecordCapacityEvent records a low-cardinality overload or quota outcome.
// Component and outcome must be controlled vocabulary values, never IDs.
func (r *Registry) RecordCapacityEvent(component, outcome string) {
	if r == nil || component == "" || outcome == "" {
		return
	}
	key := component + "\x00" + outcome
	r.mu.Lock()
	r.capacityEvents[key]++
	r.mu.Unlock()
}

func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(r.Render()))
	})
}

// CapacitySummary provides a bounded, operator-oriented view of process-local
// capacity outcomes. Counts are cumulative since process start; alerting
// systems should derive rates from the Prometheus counter instead.
func (r *Registry) CapacitySummary(now time.Time) CapacitySummary {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if r == nil {
		return CapacitySummary{GeneratedAt: now, Health: "unknown", Events: []CapacityEvent{}}
	}
	r.mu.RLock()
	keys := make([]string, 0, len(r.capacityEvents))
	for key := range r.capacityEvents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	events := make([]CapacityEvent, 0, len(keys))
	for _, key := range keys {
		parts := strings.Split(key, "\x00")
		severity, recommendation := capacityGuidance(parts[0], parts[1])
		events = append(events, CapacityEvent{
			Component: parts[0], Outcome: parts[1], Count: r.capacityEvents[key],
			Severity: severity, Recommendation: recommendation,
		})
	}
	startedAt := r.startedAt
	r.mu.RUnlock()
	health := "healthy"
	for _, event := range events {
		if event.Severity == "critical" {
			health = "critical"
			break
		}
		if event.Severity == "warning" {
			health = "attention"
		}
	}
	return CapacitySummary{GeneratedAt: now, ProcessStartedAt: startedAt, Health: health, Events: events}
}

func (r *Registry) CapacityHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(r.CapacitySummary(time.Now().UTC()))
	})
}

func capacityGuidance(component, outcome string) (string, string) {
	switch component + ":" + outcome {
	case "mqtt_ingestion:shard_queue_limit":
		return "critical", "Reduce ingress pressure and inspect shard capacity before increasing queue limits."
	case "realtime:publish_failure":
		return "critical", "Inspect Redis connectivity and recovery-provider health; Pub/Sub is not durable recovery."
	case "websocket:workspace_session_limit":
		return "warning", "Review authenticated session concurrency and close stale clients before raising the workspace quota."
	case "websocket:slow_client_disconnect":
		return "warning", "Inspect client network latency and payload volume; durable events disconnect slow clients by design."
	case "websocket:device_filter_limit":
		return "warning", "Narrow device subscriptions or revise the operator query workflow."
	case "mqtt_ingestion:hot_key_limit":
		return "warning", "Inspect the hot device or gateway and its upstream publish rate before raising per-key capacity."
	case "websocket:telemetry_coalesced":
		return "info", "Telemetry was coalesced for a slow client; check the rate if this persists."
	case "websocket:duplicate_event", "realtime:duplicate_event":
		return "info", "Duplicate delivery was suppressed; inspect rate changes around reconnect or instance transitions."
	case "realtime:invalid_message":
		return "warning", "Inspect relay protocol compatibility and reject malformed publishers."
	default:
		return "info", "Review this controlled capacity outcome and correlate it with the Prometheus counter rate."
	}
}

func (r *Registry) Render() string {
	if r == nil {
		return ""
	}
	r.mu.RLock()
	keys := make([]string, 0, len(r.requests))
	for key := range r.requests {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	metrics := make(map[string]requestMetric, len(r.requests))
	for key, value := range r.requests {
		metrics[key] = value
	}
	capacityKeys := make([]string, 0, len(r.capacityEvents))
	for key := range r.capacityEvents {
		capacityKeys = append(capacityKeys, key)
	}
	sort.Strings(capacityKeys)
	capacityEvents := make(map[string]uint64, len(r.capacityEvents))
	for key, value := range r.capacityEvents {
		capacityEvents[key] = value
	}
	startedAt := r.startedAt
	r.mu.RUnlock()

	var builder strings.Builder
	builder.WriteString("# HELP opendroneops_process_start_time_seconds Unix time when this process started.\n")
	builder.WriteString("# TYPE opendroneops_process_start_time_seconds gauge\n")
	fmt.Fprintf(&builder, "opendroneops_process_start_time_seconds %.3f\n", float64(startedAt.UnixMilli())/1000)
	builder.WriteString("# HELP opendroneops_http_requests_total HTTP requests by route and status.\n")
	builder.WriteString("# TYPE opendroneops_http_requests_total counter\n")
	builder.WriteString("# HELP opendroneops_http_request_duration_seconds HTTP request duration by route and status.\n")
	builder.WriteString("# TYPE opendroneops_http_request_duration_seconds summary\n")
	for _, key := range keys {
		parts := strings.Split(key, "\x00")
		labels := fmt.Sprintf(`method=%q,route=%q,status=%q`, parts[0], parts[1], parts[2])
		metric := metrics[key]
		fmt.Fprintf(&builder, "opendroneops_http_requests_total{%s} %d\n", labels, metric.count)
		fmt.Fprintf(&builder, "opendroneops_http_request_duration_seconds_sum{%s} %.9f\n", labels, metric.sum)
		fmt.Fprintf(&builder, "opendroneops_http_request_duration_seconds_count{%s} %d\n", labels, metric.count)
	}
	builder.WriteString("# HELP opendroneops_capacity_events_total Capacity, quota, and overload outcomes by component.\n")
	builder.WriteString("# TYPE opendroneops_capacity_events_total counter\n")
	for _, key := range capacityKeys {
		parts := strings.Split(key, "\x00")
		fmt.Fprintf(&builder, "opendroneops_capacity_events_total{component=%q,outcome=%q} %d\n", parts[0], parts[1], capacityEvents[key])
	}
	return builder.String()
}
