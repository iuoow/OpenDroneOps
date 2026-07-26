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
	rendered := registry.Render()
	if !strings.Contains(rendered, `opendroneops_http_requests_total{method="GET",route="/api/v1/health/live",status="200"} 2`) {
		t.Fatalf("metrics missing request counter: %s", rendered)
	}
	if !strings.Contains(rendered, "opendroneops_http_request_duration_seconds_count") {
		t.Fatalf("metrics missing duration count: %s", rendered)
	}
}
