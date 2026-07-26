package trajectory

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalizeQueryRejectsUnboundedRequests(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	_, err := NormalizeQuery(Query{
		WorkspaceID: "workspace-1",
		DeviceID:    "device-1",
		From:        now.Add(-25 * time.Hour),
		To:          now,
	}, now)
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected invalid query, got %v", err)
	}

	_, err = NormalizeQuery(Query{
		WorkspaceID: "workspace-1",
		DeviceID:    "device-1",
		From:        now.Add(-time.Hour),
		To:          now,
		Limit:       MaxLimit + 1,
	}, now)
	if !errors.Is(err, ErrInvalidQuery) {
		t.Fatalf("expected invalid limit, got %v", err)
	}
}

func TestMemoryStoreUsesStableCursor(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	points := make([]Point, 0, 3)
	for i, id := range []string{"a", "b", "c"} {
		points = append(points, Point{
			ID:          "00000000-0000-0000-0000-00000000000" + id,
			WorkspaceID: "workspace-1",
			DeviceID:    "device-1",
			OccurredAt:  now.Add(time.Duration(i) * time.Minute),
			ReceivedAt:  now.Add(time.Duration(i) * time.Minute),
			Latitude:    22.5 + float64(i)/1000,
			Longitude:   113.9,
		})
	}
	store := NewMemoryStore(points)
	first, err := store.Query(context.Background(), Query{
		WorkspaceID: "workspace-1", DeviceID: "device-1",
		From: now.Add(-time.Minute), To: now.Add(time.Hour), Limit: 2,
	})
	if err != nil || len(first.Items) != 2 || !first.Truncated || first.NextCursor == "" {
		t.Fatalf("first page = %+v, err=%v", first, err)
	}
	second, err := store.Query(context.Background(), Query{
		WorkspaceID: "workspace-1", DeviceID: "device-1",
		From: now.Add(-time.Minute), To: now.Add(time.Hour), Limit: 2, Cursor: first.NextCursor,
	})
	if err != nil || len(second.Items) != 1 || second.Items[0].ID != points[2].ID || second.Truncated {
		t.Fatalf("second page = %+v, err=%v", second, err)
	}
}

func TestCursorRoundTrip(t *testing.T) {
	point := Point{ID: "00000000-0000-0000-0000-000000000001", OccurredAt: time.Date(2026, 7, 26, 12, 0, 0, 123, time.UTC)}
	occurredAt, id, err := DecodeCursor(EncodeCursor(point))
	if err != nil || !occurredAt.Equal(point.OccurredAt) || id != point.ID {
		t.Fatalf("cursor = %v %q err=%v", occurredAt, id, err)
	}
}
