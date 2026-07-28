package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/observability"
)

func TestServerAddsSecurityRequestIDAndMetrics(t *testing.T) {
	server := New(":0", slog.Default())
	request := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	request.Header.Set("X-Request-ID", "request-123")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("X-Request-ID") != "request-123" || response.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing response protections: %v", response.Header())
	}
	if !strings.Contains(server.Metrics().Render(), `route="/api/v1/health/live"`) {
		t.Fatalf("request not included in metrics")
	}
}

func TestReadyReportsFailedDependency(t *testing.T) {
	server := NewWithOptions(":0", slog.Default(), []Option{
		WithReadinessProbe("postgres", func(context.Context) error { return errors.New("offline") }),
	})
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil))
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(response.Body.String(), "postgres") {
		t.Fatalf("unexpected readiness response: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestCapacitySummaryIsAdminOnly(t *testing.T) {
	registry := observability.NewRegistry(time.Now().UTC())
	registry.RecordCapacityEvent("mqtt_ingestion", "shard_queue_limit")
	server := NewWithOptions(":0", slog.Default(), []Option{
		WithAdminAddress("127.0.0.1:0"), WithMetrics(registry),
	})
	publicResponse := httptest.NewRecorder()
	server.Handler().ServeHTTP(publicResponse, httptest.NewRequest(http.MethodGet, "/capacity", nil))
	if publicResponse.Code != http.StatusNotFound {
		t.Fatalf("public capacity status = %d, want 404", publicResponse.Code)
	}
	adminResponse := httptest.NewRecorder()
	server.AdminHandler().ServeHTTP(adminResponse, httptest.NewRequest(http.MethodGet, "/capacity", nil))
	if adminResponse.Code != http.StatusOK || adminResponse.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("admin capacity response = %d headers=%v", adminResponse.Code, adminResponse.Header())
	}
	var summary observability.CapacitySummary
	if err := json.Unmarshal(adminResponse.Body.Bytes(), &summary); err != nil {
		t.Fatalf("decode capacity summary: %v", err)
	}
	if summary.Health != "critical" || len(summary.Events) != 1 {
		t.Fatalf("capacity summary = %+v", summary)
	}
}
