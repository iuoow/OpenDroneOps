package trajectory

import (
	"context"
	"sync"
	"time"
)

type Store interface {
	Query(context.Context, Query) (Page, error)
}

type MemoryStore struct {
	mu     sync.RWMutex
	points []Point
}

func NewMemoryStore(points []Point) *MemoryStore {
	copyPoints := append([]Point(nil), points...)
	sortPoints(copyPoints)
	return &MemoryStore{points: copyPoints}
}

func (s *MemoryStore) Query(ctx context.Context, query Query) (Page, error) {
	if err := ctx.Err(); err != nil {
		return Page{}, err
	}
	normalized, err := NormalizeQuery(query, query.To)
	if err != nil {
		return Page{}, err
	}
	var afterOccurredAt time.Time
	var afterID string
	if normalized.Cursor != "" {
		occurredAt, id, err := DecodeCursor(normalized.Cursor)
		if err != nil {
			return Page{}, err
		}
		afterOccurredAt, afterID = occurredAt, id
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]Point, 0, normalized.Limit+1)
	for _, point := range s.points {
		if point.WorkspaceID != normalized.WorkspaceID || point.DeviceID != normalized.DeviceID ||
			point.OccurredAt.Before(normalized.From) || !point.OccurredAt.Before(normalized.To) {
			continue
		}
		if !afterOccurredAt.IsZero() {
			if point.OccurredAt.Before(afterOccurredAt) ||
				(point.OccurredAt.Equal(afterOccurredAt) && point.ID <= afterID) {
				continue
			}
		}
		items = append(items, point)
		if len(items) == normalized.Limit+1 {
			break
		}
	}
	page := Page{Items: items, Truncated: len(items) > normalized.Limit}
	if page.Truncated {
		page.Items = page.Items[:normalized.Limit]
		page.NextCursor = EncodeCursor(page.Items[len(page.Items)-1])
	}
	return page, nil
}
