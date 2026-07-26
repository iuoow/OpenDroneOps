package alarm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	if db == nil {
		return nil, errors.New("postgres database is required")
	}
	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) UpsertFinding(ctx context.Context, finding Finding) (domain.Alarm, bool, error) {
	if err := validateFinding(finding); err != nil {
		return domain.Alarm{}, false, err
	}
	details := finding.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	const query = `
INSERT INTO alarms (
  workspace_id, device_id, dedup_key, alarm_type, severity, status,
  first_occurred_at, last_occurred_at, occurrence_count, details, updated_at
) VALUES ($1, $2, $3, $4, $5, 'OPEN', $6, $6, 1, $7, now())
ON CONFLICT (workspace_id, dedup_key) WHERE status IN ('OPEN', 'ACKNOWLEDGED')
DO UPDATE SET
  last_occurred_at = GREATEST(alarms.last_occurred_at, EXCLUDED.last_occurred_at),
  occurrence_count = alarms.occurrence_count + 1,
  severity = CASE
    WHEN alarms.severity = 'CRITICAL' OR EXCLUDED.severity = 'CRITICAL' THEN 'CRITICAL'
    WHEN alarms.severity = 'WARNING' OR EXCLUDED.severity = 'WARNING' THEN 'WARNING'
    ELSE 'INFO'
  END,
  details = EXCLUDED.details,
  updated_at = now()
RETURNING id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
          first_occurred_at, last_occurred_at, occurrence_count,
          COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details,
          (xmax = 0) AS created`
	var alarm domain.Alarm
	var acknowledgedBy string
	var acknowledgedAt, resolvedAt sql.NullTime
	var detailsBytes []byte
	var created bool
	err := s.db.QueryRowContext(ctx, query,
		finding.WorkspaceID, finding.DeviceID, finding.DedupKey, finding.AlarmType,
		finding.Severity, finding.OccurredAt, details,
	).Scan(
		&alarm.ID, &alarm.WorkspaceID, &alarm.DeviceID, &alarm.DedupKey, &alarm.AlarmType,
		&alarm.Severity, &alarm.Status, &alarm.FirstOccurredAt, &alarm.LastOccurredAt,
		&alarm.OccurrenceCount, &acknowledgedBy, &acknowledgedAt, &resolvedAt, &detailsBytes,
		&created,
	)
	if err != nil {
		return domain.Alarm{}, false, fmt.Errorf("upsert alarm finding: %w", err)
	}
	alarm.AcknowledgedBy = acknowledgedBy
	alarm.AcknowledgedAt = nullTimePtr(acknowledgedAt)
	alarm.ResolvedAt = nullTimePtr(resolvedAt)
	alarm.Details = json.RawMessage(detailsBytes)
	return alarm, created, nil
}

func (s *PostgresStore) Acknowledge(ctx context.Context, workspaceID, alarmID domain.ID, actor string, at time.Time) (domain.Alarm, bool, error) {
	if actor == "" {
		return domain.Alarm{}, false, fmt.Errorf("%w: acknowledge requires actor", domain.ErrInvalidEntity)
	}
	const query = `
UPDATE alarms
SET status = 'ACKNOWLEDGED', acknowledged_by = $3, acknowledged_at = $4, updated_at = now()
WHERE workspace_id = $1 AND id = $2 AND status = 'OPEN'
RETURNING id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
          first_occurred_at, last_occurred_at, occurrence_count,
          COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details`
	alarm, err := s.queryAlarm(ctx, query, workspaceID, alarmID, actor, at)
	if err == nil {
		return alarm, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Alarm{}, false, err
	}
	alarm, err = s.Get(ctx, workspaceID, alarmID)
	if err != nil {
		return domain.Alarm{}, false, err
	}
	if alarm.Status == domain.AlarmAcknowledged {
		return alarm, false, nil
	}
	return domain.Alarm{}, false, fmt.Errorf("%w: alarm cannot be acknowledged", domain.ErrInvalidTransition)
}

func (s *PostgresStore) Resolve(ctx context.Context, workspaceID, alarmID domain.ID, at time.Time) (domain.Alarm, bool, error) {
	const query = `
UPDATE alarms
SET status = 'RESOLVED', resolved_at = $3, updated_at = now()
WHERE workspace_id = $1 AND id = $2 AND status IN ('OPEN', 'ACKNOWLEDGED')
RETURNING id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
          first_occurred_at, last_occurred_at, occurrence_count,
          COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details`
	alarm, err := s.queryAlarm(ctx, query, workspaceID, alarmID, at)
	if err == nil {
		return alarm, true, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return domain.Alarm{}, false, err
	}
	alarm, err = s.Get(ctx, workspaceID, alarmID)
	if err != nil {
		return domain.Alarm{}, false, err
	}
	if alarm.Status == domain.AlarmResolved {
		return alarm, false, nil
	}
	return domain.Alarm{}, false, fmt.Errorf("%w: alarm cannot be resolved", domain.ErrInvalidTransition)
}

func (s *PostgresStore) ResolveByDedup(ctx context.Context, resolution Resolution) (domain.Alarm, bool, error) {
	if resolution.WorkspaceID == "" || resolution.DeviceID == "" || resolution.DedupKey == "" {
		return domain.Alarm{}, false, nil
	}
	const query = `
UPDATE alarms
SET status = 'RESOLVED', resolved_at = $4, updated_at = now()
WHERE workspace_id = $1 AND device_id = $2 AND dedup_key = $3
  AND status IN ('OPEN', 'ACKNOWLEDGED')
RETURNING id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
          first_occurred_at, last_occurred_at, occurrence_count,
          COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details`
	alarm, err := s.queryAlarm(ctx, query, resolution.WorkspaceID, resolution.DeviceID, resolution.DedupKey, resolution.OccurredAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Alarm{}, false, nil
	}
	if err != nil {
		return domain.Alarm{}, false, err
	}
	return alarm, true, nil
}

func (s *PostgresStore) Get(ctx context.Context, workspaceID, alarmID domain.ID) (domain.Alarm, error) {
	const query = `
SELECT id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
       first_occurred_at, last_occurred_at, occurrence_count,
       COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details
FROM alarms
WHERE workspace_id = $1 AND id = $2`
	alarm, err := s.queryAlarm(ctx, query, workspaceID, alarmID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Alarm{}, ErrAlarmNotFound
	}
	if err != nil {
		return domain.Alarm{}, err
	}
	return alarm, nil
}

func (s *PostgresStore) ListActive(ctx context.Context, workspaceID domain.ID) ([]domain.Alarm, error) {
	const query = `
SELECT id, workspace_id, device_id, dedup_key, alarm_type, severity, status,
       first_occurred_at, last_occurred_at, occurrence_count,
       COALESCE(acknowledged_by, ''), acknowledged_at, resolved_at, details
FROM alarms
WHERE workspace_id = $1 AND status IN ('OPEN', 'ACKNOWLEDGED')
ORDER BY last_occurred_at DESC, id DESC`
	rows, err := s.db.QueryContext(ctx, query, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list active alarms: %w", err)
	}
	defer rows.Close()
	var alarms []domain.Alarm
	for rows.Next() {
		alarm, err := scanAlarm(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active alarm: %w", err)
		}
		alarms = append(alarms, alarm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active alarms: %w", err)
	}
	return alarms, nil
}

type rowScanner interface {
	Scan(...any) error
}

func (s *PostgresStore) queryAlarm(ctx context.Context, query string, args ...any) (domain.Alarm, error) {
	return scanAlarm(s.db.QueryRowContext(ctx, query, args...))
}

func scanAlarm(row rowScanner) (domain.Alarm, error) {
	var alarm domain.Alarm
	var acknowledgedBy string
	var acknowledgedAt, resolvedAt sql.NullTime
	var details []byte
	err := row.Scan(
		&alarm.ID, &alarm.WorkspaceID, &alarm.DeviceID, &alarm.DedupKey, &alarm.AlarmType,
		&alarm.Severity, &alarm.Status, &alarm.FirstOccurredAt, &alarm.LastOccurredAt,
		&alarm.OccurrenceCount, &acknowledgedBy, &acknowledgedAt, &resolvedAt, &details,
	)
	if err != nil {
		return domain.Alarm{}, err
	}
	alarm.AcknowledgedBy = acknowledgedBy
	alarm.AcknowledgedAt = nullTimePtr(acknowledgedAt)
	alarm.ResolvedAt = nullTimePtr(resolvedAt)
	alarm.Details = json.RawMessage(details)
	return alarm, nil
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
