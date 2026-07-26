package trajectory

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestHandlerRequiresWorkspaceAndReturnsPageMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore([]Point{{
		ID: "point-1", WorkspaceID: "workspace-1", DeviceID: "device-1",
		OccurredAt: now.Add(-time.Minute), ReceivedAt: now.Add(-time.Minute),
		Latitude: 22.5, Longitude: 113.9,
	}})
	handler, err := NewHandler(store)
	if err != nil {
		t.Fatal(err)
	}
	handler.now = func() time.Time { return now }
	router := gin.New()
	api := router.Group("/api/v1")
	handler.Register(api)

	request := httptest.NewRequest("GET", "/api/v1/devices/device-1/trajectory?from="+
		now.Add(-time.Hour).Format(time.RFC3339)+"&to="+now.Format(time.RFC3339), nil)
	request.Header.Set("X-Workspace-ID", "workspace-1")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 200 || !strings.Contains(response.Body.String(), `"truncated":false`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	request = httptest.NewRequest("GET", "/api/v1/devices/device-1/trajectory", nil)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatalf("missing workspace status=%d", response.Code)
	}
}
