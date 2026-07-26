package observability

import (
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

// Registry is a deliberately small Prometheus text-format registry. It keeps
// the HTTP boundary observable without introducing an exporter dependency.
type Registry struct {
	mu        sync.RWMutex
	startedAt time.Time
	requests  map[string]requestMetric
}

func NewRegistry(now time.Time) *Registry {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &Registry{startedAt: now, requests: make(map[string]requestMetric)}
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

func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(r.Render()))
	})
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
	return builder.String()
}
