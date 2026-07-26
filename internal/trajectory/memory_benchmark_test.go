package trajectory

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func BenchmarkMemoryStoreQuery5000Points(b *testing.B) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	points := make([]Point, 5000)
	for index := range points {
		points[index] = Point{
			ID:          fmt.Sprintf("point-%05d", index),
			WorkspaceID: "workspace-1",
			DeviceID:    "device-1",
			OccurredAt:  now.Add(time.Duration(index) * time.Millisecond),
			ReceivedAt:  now.Add(time.Duration(index) * time.Millisecond),
		}
	}
	store := NewMemoryStore(points)
	query := Query{WorkspaceID: "workspace-1", DeviceID: "device-1", From: now.Add(-time.Second), To: now.Add(time.Hour), Limit: MaxLimit}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		if _, err := store.Query(context.Background(), query); err != nil {
			b.Fatal(err)
		}
	}
}
