package twin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) (*PostgresRepository, error) {
	if db == nil {
		return nil, fmt.Errorf("postgres database is required")
	}
	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) UpsertLatest(ctx context.Context, state domain.DeviceState) (bool, error) {
	payload := state.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	const query = `
INSERT INTO device_latest_states (
  device_id, workspace_id, state_version, device_time, server_time, online,
  latitude, longitude, altitude, battery_percent, mode, payload, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, now())
ON CONFLICT (device_id) DO UPDATE SET
  workspace_id = EXCLUDED.workspace_id,
  state_version = EXCLUDED.state_version,
  device_time = EXCLUDED.device_time,
  server_time = EXCLUDED.server_time,
  online = EXCLUDED.online,
  latitude = EXCLUDED.latitude,
  longitude = EXCLUDED.longitude,
  altitude = EXCLUDED.altitude,
  battery_percent = EXCLUDED.battery_percent,
  mode = EXCLUDED.mode,
  payload = EXCLUDED.payload,
  updated_at = now()
WHERE device_latest_states.workspace_id = EXCLUDED.workspace_id
  AND device_latest_states.state_version < EXCLUDED.state_version`
	result, err := r.db.ExecContext(ctx, query,
		state.DeviceID, state.WorkspaceID, state.StateVersion, state.DeviceTime, state.ServerTime,
		state.Online, state.Latitude, state.Longitude, state.Altitude, state.BatteryPercent,
		state.Mode, payload,
	)
	if err != nil {
		return false, fmt.Errorf("upsert latest state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read latest state update count: %w", err)
	}
	return rows == 1, nil
}

func (r *PostgresRepository) GetLatest(ctx context.Context, workspaceID, deviceID domain.ID) (domain.DeviceState, error) {
	const query = `
SELECT device_id, workspace_id, state_version, device_time, server_time, online,
       latitude, longitude, altitude, battery_percent, mode, payload
FROM device_latest_states
WHERE workspace_id = $1 AND device_id = $2`
	var state domain.DeviceState
	var payload []byte
	err := r.db.QueryRowContext(ctx, query, workspaceID, deviceID).Scan(
		&state.DeviceID, &state.WorkspaceID, &state.StateVersion, &state.DeviceTime,
		&state.ServerTime, &state.Online, &state.Latitude, &state.Longitude,
		&state.Altitude, &state.BatteryPercent, &state.Mode, &payload,
	)
	if err != nil {
		return domain.DeviceState{}, fmt.Errorf("get latest state: %w", err)
	}
	state.Payload = json.RawMessage(payload)
	return state, nil
}

func (r *PostgresRepository) AppendEvent(ctx context.Context, event domain.DeviceEvent) (bool, error) {
	payload := event.Payload
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	const query = `
INSERT INTO device_events (
  event_id, workspace_id, device_id, gateway_device_id, event_type, method,
  device_time, received_at, sequence, payload
) VALUES ($1, $2, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid, $5, NULLIF($6, ''),
          $7, $8, $9, $10)
ON CONFLICT (workspace_id, event_id) DO NOTHING`
	result, err := r.db.ExecContext(ctx, query,
		event.EventID, event.WorkspaceID, event.DeviceID, event.GatewayDeviceID,
		event.EventType, event.Method, event.DeviceTime, event.ReceivedAt,
		event.Sequence, payload,
	)
	if err != nil {
		return false, fmt.Errorf("append device event: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read device event insert count: %w", err)
	}
	return rows == 1, nil
}
