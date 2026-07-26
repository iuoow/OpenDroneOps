-- +goose Up
ALTER TABLE alarms
  ADD CONSTRAINT alarms_severity_check
    CHECK (severity IN ('INFO', 'WARNING', 'CRITICAL')),
  ADD CONSTRAINT alarms_status_check
    CHECK (status IN ('OPEN', 'ACKNOWLEDGED', 'RESOLVED')),
  ADD CONSTRAINT alarms_occurrence_count_check
    CHECK (occurrence_count > 0);

-- +goose Down
ALTER TABLE alarms
  DROP CONSTRAINT IF EXISTS alarms_occurrence_count_check,
  DROP CONSTRAINT IF EXISTS alarms_status_check,
  DROP CONSTRAINT IF EXISTS alarms_severity_check;
