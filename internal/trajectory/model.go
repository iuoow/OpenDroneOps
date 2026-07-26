package trajectory

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

const (
	DefaultLimit = 500
	MaxLimit     = 5000
	MaxWindow    = 24 * time.Hour
)

var (
	ErrInvalidQuery  = errors.New("invalid trajectory query")
	ErrInvalidCursor = errors.New("invalid trajectory cursor")
)

type Point struct {
	ID             string    `json:"id"`
	WorkspaceID    string    `json:"workspace_id"`
	DeviceID       string    `json:"device_id"`
	OccurredAt     time.Time `json:"occurred_at"`
	ReceivedAt     time.Time `json:"received_at"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	Altitude       *float64  `json:"altitude,omitempty"`
	Speed          *float64  `json:"speed,omitempty"`
	Heading        *float64  `json:"heading,omitempty"`
	BatteryPercent *float64  `json:"battery_percent,omitempty"`
	SourceEventID  string    `json:"source_event_id,omitempty"`
}

type Query struct {
	WorkspaceID string
	DeviceID    string
	From        time.Time
	To          time.Time
	Limit       int
	Cursor      string
}

type Page struct {
	Items      []Point `json:"items"`
	NextCursor string  `json:"next_cursor,omitempty"`
	Truncated  bool    `json:"truncated"`
}

type cursor struct {
	OccurredAt time.Time `json:"occurred_at"`
	ID         string    `json:"id"`
}

func NormalizeQuery(query Query, now time.Time) (Query, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if query.WorkspaceID == "" || query.DeviceID == "" {
		return Query{}, fmt.Errorf("%w: workspace and device are required", ErrInvalidQuery)
	}
	if query.From.IsZero() {
		query.From = now.Add(-time.Hour)
	}
	if query.To.IsZero() {
		query.To = now
	}
	query.From = query.From.UTC()
	query.To = query.To.UTC()
	if !query.From.Before(query.To) {
		return Query{}, fmt.Errorf("%w: from must be before to", ErrInvalidQuery)
	}
	if query.To.Sub(query.From) > MaxWindow {
		return Query{}, fmt.Errorf("%w: time window exceeds %s", ErrInvalidQuery, MaxWindow)
	}
	if query.Limit == 0 {
		query.Limit = DefaultLimit
	}
	if query.Limit < 1 || query.Limit > MaxLimit {
		return Query{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidQuery, MaxLimit)
	}
	if query.Cursor != "" {
		if _, err := decodeCursor(query.Cursor); err != nil {
			return Query{}, err
		}
	}
	return query, nil
}

func EncodeCursor(point Point) string {
	raw, _ := json.Marshal(cursor{OccurredAt: point.OccurredAt.UTC(), ID: point.ID})
	return base64.RawURLEncoding.EncodeToString(raw)
}

func decodeCursor(value string) (cursor, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return cursor{}, fmt.Errorf("%w: base64", ErrInvalidCursor)
	}
	var result cursor
	if err := json.Unmarshal(raw, &result); err != nil || result.ID == "" || result.OccurredAt.IsZero() {
		return cursor{}, ErrInvalidCursor
	}
	result.OccurredAt = result.OccurredAt.UTC()
	return result, nil
}

func DecodeCursor(value string) (occurredAt time.Time, id string, err error) {
	result, err := decodeCursor(value)
	if err != nil {
		return time.Time{}, "", err
	}
	return result.OccurredAt, result.ID, nil
}

func sortPoints(points []Point) {
	sort.Slice(points, func(i, j int) bool {
		if points[i].OccurredAt.Equal(points[j].OccurredAt) {
			return points[i].ID < points[j].ID
		}
		return points[i].OccurredAt.Before(points[j].OccurredAt)
	})
}
