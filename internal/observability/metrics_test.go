package observability

import (
	"strings"
	"testing"
	"time"
)

func TestRegistryRendersStablePrometheusMetrics(t *testing.T) {
	registry := NewRegistry(time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC))
	registry.RecordHTTP("GET", "/api/v1/health/live", 200, 12*time.Millisecond)
	registry.RecordHTTP("GET", "/api/v1/health/live", 200, 8*time.Millisecond)
	registry.RecordCapacityEvent("websocket", "workspace_session_limit")
	rendered := registry.Render()
	if !strings.Contains(rendered, `opendroneops_http_requests_total{method="GET",route="/api/v1/health/live",status="200"} 2`) {
		t.Fatalf("metrics missing request counter: %s", rendered)
	}
	if !strings.Contains(rendered, "opendroneops_http_request_duration_seconds_count") {
		t.Fatalf("metrics missing duration count: %s", rendered)
	}
	if !strings.Contains(rendered, `opendroneops_capacity_events_total{component="websocket",outcome="workspace_session_limit"} 1`) {
		t.Fatalf("metrics missing capacity counter: %s", rendered)
	}
}

func TestCapacitySummaryProvidesOrderedGuidance(t *testing.T) {
	registry := NewRegistry(time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC))
	registry.RecordCapacityEvent("websocket", "telemetry_coalesced")
	registry.RecordCapacityEvent("mqtt_ingestion", "shard_queue_limit")
	summary := registry.CapacitySummary(time.Date(2026, 7, 28, 1, 0, 0, 0, time.UTC))
	if summary.Health != "critical" || len(summary.Events) != 2 {
		t.Fatalf("CapacitySummary() = %+v", summary)
	}
	if summary.Events[0].Component != "mqtt_ingestion" || summary.Events[0].Severity != "critical" {
		t.Fatalf("unexpected ordered capacity event: %+v", summary.Events)
	}
	if summary.Events[1].Severity != "info" || summary.Events[1].Recommendation == "" {
		t.Fatalf("missing operator guidance: %+v", summary.Events[1])
	}
}
