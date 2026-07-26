package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
