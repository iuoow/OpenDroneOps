package trajectory

import (
	"context"
	"database/sql"
	"fmt"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, fmt.Errorf("postgres database is required")
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) Query(ctx context.Context, query Query) (Page, error) {
	normalized, err := NormalizeQuery(query, query.To)
	if err != nil {
		return Page{}, err
	}
	cursorTime, cursorID := normalized.From, ""
	if normalized.Cursor != "" {
		cursorTime, cursorID, err = DecodeCursor(normalized.Cursor)
		if err != nil {
			return Page{}, err
		}
	}
	const statement = `
SELECT id::text, workspace_id::text, device_id::text, occurred_at, received_at,
       latitude, longitude, altitude, speed, heading, battery_percent,
       COALESCE(source_event_id::text, '')
FROM trajectory_points
WHERE workspace_id = $1::uuid
  AND device_id = $2::uuid
  AND occurred_at >= $3
  AND occurred_at < $4
  AND ($5 = '' OR (occurred_at, id) > ($6, $5::uuid))
ORDER BY occurred_at ASC, id ASC
LIMIT $7`
	rows, err := s.db.QueryContext(ctx, statement, normalized.WorkspaceID, normalized.DeviceID,
		normalized.From, normalized.To, cursorID, cursorTime, normalized.Limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("query trajectory: %w", err)
	}
	defer rows.Close()
	items := make([]Point, 0, normalized.Limit+1)
	for rows.Next() {
		var point Point
		if err := rows.Scan(&point.ID, &point.WorkspaceID, &point.DeviceID, &point.OccurredAt,
			&point.ReceivedAt, &point.Latitude, &point.Longitude, &point.Altitude, &point.Speed,
			&point.Heading, &point.BatteryPercent, &point.SourceEventID); err != nil {
			return Page{}, fmt.Errorf("scan trajectory: %w", err)
		}
		items = append(items, point)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("iterate trajectory: %w", err)
	}
	page := Page{Items: items, Truncated: len(items) > normalized.Limit}
	if page.Truncated {
		page.Items = page.Items[:normalized.Limit]
		page.NextCursor = EncodeCursor(page.Items[len(page.Items)-1])
	}
	return page, nil
}
