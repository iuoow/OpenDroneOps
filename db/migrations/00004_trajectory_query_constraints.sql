-- +goose Up
ALTER TABLE trajectory_points
  ADD COLUMN battery_percent DOUBLE PRECISION,
  ADD CONSTRAINT trajectory_latitude_check CHECK (latitude BETWEEN -90 AND 90),
  ADD CONSTRAINT trajectory_longitude_check CHECK (longitude BETWEEN -180 AND 180),
  ADD CONSTRAINT trajectory_speed_check CHECK (speed IS NULL OR speed >= 0),
  ADD CONSTRAINT trajectory_heading_check CHECK (heading IS NULL OR (heading >= 0 AND heading < 360)),
  ADD CONSTRAINT trajectory_battery_check CHECK (battery_percent IS NULL OR battery_percent BETWEEN 0 AND 100);

CREATE UNIQUE INDEX trajectory_source_event_unique
ON trajectory_points(workspace_id, device_id, source_event_id)
WHERE source_event_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS trajectory_source_event_unique;

ALTER TABLE trajectory_points
  DROP CONSTRAINT IF EXISTS trajectory_battery_check,
  DROP CONSTRAINT IF EXISTS trajectory_heading_check,
  DROP CONSTRAINT IF EXISTS trajectory_speed_check,
  DROP CONSTRAINT IF EXISTS trajectory_longitude_check,
  DROP CONSTRAINT IF EXISTS trajectory_latitude_check,
  DROP COLUMN IF EXISTS battery_percent;
