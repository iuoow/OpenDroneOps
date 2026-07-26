-- Reference blueprint: the executable schema is db/migrations/00001_initial.sql.
-- Keep this file as a readable schema reference; apply Goose migrations in deployments.
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE workspaces (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE devices (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  vendor TEXT NOT NULL,
  serial_number TEXT NOT NULL,
  gateway_serial_number TEXT,
  product_model TEXT,
  device_type TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'REGISTERED',
  capabilities JSONB NOT NULL DEFAULT '{}'::jsonb,
  registered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, vendor, serial_number)
);
CREATE INDEX devices_workspace_status_idx ON devices(workspace_id, status, id);

CREATE TABLE device_relationships (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  parent_device_id UUID NOT NULL REFERENCES devices(id),
  child_device_id UUID NOT NULL REFERENCES devices(id),
  relationship_type TEXT NOT NULL,
  valid_from TIMESTAMPTZ NOT NULL DEFAULT now(),
  valid_to TIMESTAMPTZ,
  CHECK (parent_device_id <> child_device_id)
);
CREATE UNIQUE INDEX active_device_relationship_unique
ON device_relationships(workspace_id, parent_device_id, child_device_id, relationship_type)
WHERE valid_to IS NULL;

CREATE TABLE raw_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID REFERENCES workspaces(id),
  direction TEXT NOT NULL,
  topic TEXT NOT NULL,
  qos SMALLINT,
  payload_hash BYTEA NOT NULL,
  payload JSONB,
  received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  retention_class TEXT NOT NULL DEFAULT 'NORMAL',
  redacted BOOLEAN NOT NULL DEFAULT false
);
CREATE INDEX raw_messages_received_idx ON raw_messages(received_at DESC);

CREATE TABLE processed_messages (
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  dedup_key TEXT NOT NULL,
  first_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  raw_message_id UUID REFERENCES raw_messages(id),
  PRIMARY KEY (workspace_id, dedup_key)
);

CREATE TABLE device_latest_states (
  device_id UUID PRIMARY KEY REFERENCES devices(id),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  state_version BIGINT NOT NULL,
  device_time TIMESTAMPTZ,
  server_time TIMESTAMPTZ NOT NULL,
  online BOOLEAN NOT NULL,
  latitude DOUBLE PRECISION,
  longitude DOUBLE PRECISION,
  altitude DOUBLE PRECISION,
  battery_percent DOUBLE PRECISION,
  mode TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (battery_percent IS NULL OR battery_percent BETWEEN 0 AND 100)
);
CREATE INDEX device_latest_states_workspace_online_idx
ON device_latest_states(workspace_id, online, updated_at DESC);

CREATE TABLE device_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  event_id TEXT NOT NULL,
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  device_id UUID REFERENCES devices(id),
  gateway_device_id UUID REFERENCES devices(id),
  event_type TEXT NOT NULL,
  method TEXT,
  device_time TIMESTAMPTZ,
  received_at TIMESTAMPTZ NOT NULL,
  sequence BIGINT,
  raw_message_id UUID REFERENCES raw_messages(id),
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  UNIQUE (workspace_id, event_id)
);
CREATE INDEX device_events_device_time_idx
ON device_events(workspace_id, device_id, received_at DESC, id DESC);

CREATE TABLE trajectory_points (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  device_id UUID NOT NULL REFERENCES devices(id),
  occurred_at TIMESTAMPTZ NOT NULL,
  received_at TIMESTAMPTZ NOT NULL,
  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  altitude DOUBLE PRECISION,
  speed DOUBLE PRECISION,
  heading DOUBLE PRECISION,
  source_event_id UUID REFERENCES device_events(id)
);
CREATE INDEX trajectory_device_time_idx
ON trajectory_points(workspace_id, device_id, occurred_at DESC, id DESC);

CREATE TABLE alarms (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  device_id UUID NOT NULL REFERENCES devices(id),
  dedup_key TEXT NOT NULL,
  alarm_type TEXT NOT NULL,
  severity TEXT NOT NULL,
  status TEXT NOT NULL,
  first_occurred_at TIMESTAMPTZ NOT NULL,
  last_occurred_at TIMESTAMPTZ NOT NULL,
  occurrence_count BIGINT NOT NULL DEFAULT 1,
  acknowledged_by TEXT,
  acknowledged_at TIMESTAMPTZ,
  resolved_at TIMESTAMPTZ,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX alarms_active_dedup_unique
ON alarms(workspace_id, dedup_key)
WHERE status IN ('OPEN', 'ACKNOWLEDGED');

CREATE TABLE commands (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  target_device_id UUID NOT NULL REFERENCES devices(id),
  gateway_device_id UUID REFERENCES devices(id),
  method TEXT NOT NULL,
  status TEXT NOT NULL,
  risk_level TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash BYTEA NOT NULL,
  dji_tid TEXT,
  dji_bid TEXT,
  parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
  requested_by TEXT NOT NULL,
  result_code INTEGER,
  result_message TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  completed_at TIMESTAMPTZ,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, idempotency_key)
);
CREATE UNIQUE INDEX commands_dji_correlation_unique
ON commands(workspace_id, dji_tid, dji_bid, method)
WHERE dji_tid IS NOT NULL AND dji_bid IS NOT NULL;

CREATE TABLE command_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  command_id UUID NOT NULL REFERENCES commands(id),
  from_status TEXT,
  to_status TEXT NOT NULL,
  source TEXT NOT NULL,
  result_code INTEGER,
  message TEXT,
  raw_message_id UUID REFERENCES raw_messages(id),
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE outbox_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  aggregate_type TEXT NOT NULL,
  aggregate_id UUID NOT NULL,
  event_type TEXT NOT NULL,
  destination TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL DEFAULT 'PENDING',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  locked_at TIMESTAMPTZ,
  locked_by TEXT,
  published_at TIMESTAMPTZ,
  last_error TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX outbox_pending_idx
ON outbox_events(status, available_at, created_at)
WHERE status IN ('PENDING', 'RETRY');

CREATE TABLE quarantine_messages (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  raw_message_id UUID REFERENCES raw_messages(id),
  error_class TEXT NOT NULL,
  error_message TEXT NOT NULL,
  first_failed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_failed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  retry_count INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'OPEN'
);

CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  actor_id TEXT NOT NULL,
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  request_id TEXT,
  details JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_workspace_time_idx ON audit_logs(workspace_id, created_at DESC, id DESC);
