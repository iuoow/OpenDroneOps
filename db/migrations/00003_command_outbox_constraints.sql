-- +goose Up
ALTER TABLE commands
  ADD CONSTRAINT commands_status_check
    CHECK (status IN (
      'CREATED', 'VALIDATED', 'REJECTED', 'PUBLISH_PENDING', 'PUBLISHED',
      'ACCEPTED', 'EXECUTING', 'SUCCEEDED', 'FAILED', 'TIMEOUT', 'CANCELED'
    )),
  ADD CONSTRAINT commands_risk_level_check
    CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH'));

ALTER TABLE outbox_events
  ADD CONSTRAINT outbox_status_check
    CHECK (status IN ('PENDING', 'PROCESSING', 'RETRY', 'PUBLISHED', 'FAILED')),
  ADD CONSTRAINT outbox_attempt_count_check
    CHECK (attempt_count >= 0);

DROP INDEX outbox_pending_idx;
CREATE INDEX outbox_pending_idx
ON outbox_events(status, available_at, locked_at, created_at)
WHERE status IN ('PENDING', 'PROCESSING', 'RETRY');

CREATE TABLE orphan_command_replies (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  workspace_id UUID NOT NULL REFERENCES workspaces(id),
  tid TEXT NOT NULL,
  bid TEXT NOT NULL,
  method TEXT NOT NULL,
  gateway_sn TEXT NOT NULL,
  payload_hash TEXT NOT NULL,
  payload JSONB NOT NULL,
  received_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (workspace_id, tid, bid, method, payload_hash)
);
CREATE INDEX orphan_command_replies_received_idx
ON orphan_command_replies(workspace_id, received_at DESC, id DESC);

-- +goose Down
DROP TABLE IF EXISTS orphan_command_replies;

DROP INDEX outbox_pending_idx;
CREATE INDEX outbox_pending_idx
ON outbox_events(status, available_at, created_at)
WHERE status IN ('PENDING', 'RETRY');

ALTER TABLE outbox_events
  DROP CONSTRAINT IF EXISTS outbox_attempt_count_check,
  DROP CONSTRAINT IF EXISTS outbox_status_check;

ALTER TABLE commands
  DROP CONSTRAINT IF EXISTS commands_risk_level_check,
  DROP CONSTRAINT IF EXISTS commands_status_check;
