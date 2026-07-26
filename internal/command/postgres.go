package command

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) (*PostgresRepository, error) {
	if db == nil {
		return nil, errors.New("postgres database is required")
	}
	return &PostgresRepository{db: db}, nil
}

func (r *PostgresRepository) Create(ctx context.Context, bundle CreateBundle) (domain.Command, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Command{}, false, err
	}
	defer tx.Rollback()
	hashBytes, err := hex.DecodeString(bundle.Command.RequestHash)
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("decode request hash: %w", err)
	}
	const insertCommand = `
INSERT INTO commands (
  id, workspace_id, target_device_id, gateway_device_id, method, status, risk_level,
  idempotency_key, request_hash, dji_tid, dji_bid, parameters, requested_by,
  created_at, expires_at, completed_at, updated_at
) VALUES ($1, $2, $3, NULLIF($4, '')::uuid, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
ON CONFLICT (workspace_id, idempotency_key) DO NOTHING`
	result, err := tx.ExecContext(ctx, insertCommand,
		bundle.Command.ID, bundle.Command.WorkspaceID, bundle.Command.TargetDeviceID, bundle.Command.GatewayDeviceID,
		bundle.Command.Method, bundle.Command.Status, bundle.Command.RiskLevel, bundle.Command.IdempotencyKey,
		hashBytes, bundle.Command.DJITID, bundle.Command.DJIBID, jsonOrObject(bundle.Command.Parameters),
		bundle.Command.RequestedBy, bundle.Command.CreatedAt, bundle.Command.ExpiresAt,
		bundle.Command.CompletedAt, bundle.Command.UpdatedAt,
	)
	if err != nil {
		return domain.Command{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.Command{}, false, err
	}
	if rows == 0 {
		existing, err := queryCommand(ctx, tx, `
SELECT `+commandColumns+` FROM commands WHERE workspace_id = $1 AND idempotency_key = $2`,
			bundle.Command.WorkspaceID, bundle.Command.IdempotencyKey)
		if err != nil {
			return domain.Command{}, false, err
		}
		if err := domain.EnsureIdempotency(existing.RequestHash, bundle.Command.RequestHash); err != nil {
			return domain.Command{}, false, err
		}
		if err := tx.Commit(); err != nil {
			return domain.Command{}, false, err
		}
		return existing, false, nil
	}
	for _, event := range bundle.Events {
		if err := insertCommandEvent(ctx, tx, event, ""); err != nil {
			return domain.Command{}, false, err
		}
	}
	const insertOutbox = `
INSERT INTO outbox_events (
  id, workspace_id, aggregate_type, aggregate_id, event_type, destination, payload,
  status, attempt_count, available_at, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	if _, err := tx.ExecContext(ctx, insertOutbox,
		bundle.Outbox.ID, bundle.Outbox.WorkspaceID, bundle.Outbox.AggregateType, bundle.Outbox.AggregateID,
		bundle.Outbox.EventType, bundle.Outbox.Destination, jsonOrObject(bundle.Outbox.Payload),
		bundle.Outbox.Status, bundle.Outbox.AttemptCount, bundle.Outbox.AvailableAt, bundle.Outbox.CreatedAt,
	); err != nil {
		return domain.Command{}, false, err
	}
	if err := insertAudit(ctx, tx, bundle.Audit); err != nil {
		return domain.Command{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Command{}, false, err
	}
	return bundle.Command, true, nil
}

func (r *PostgresRepository) LeaseOutbox(ctx context.Context, workerID string, limit int, now time.Time, leaseDuration time.Duration) ([]Delivery, error) {
	const query = `
WITH candidates AS (
  SELECT o.id
  FROM outbox_events o
  WHERE (
      o.status IN ('PENDING', 'RETRY') AND o.available_at <= $3
    ) OR (
      o.status = 'PROCESSING' AND o.locked_at <= $3 - $4::interval
    )
  ORDER BY o.available_at, o.created_at, o.id
  FOR UPDATE SKIP LOCKED
  LIMIT $2
)
UPDATE outbox_events o
SET status = 'PROCESSING', locked_at = $3, locked_by = $1, attempt_count = o.attempt_count + 1
FROM candidates
WHERE o.id = candidates.id
RETURNING o.id, o.workspace_id, o.aggregate_type, o.aggregate_id, o.event_type,
          o.destination, o.payload, o.status, o.attempt_count, o.available_at,
          o.locked_at, COALESCE(o.locked_by, ''), o.published_at,
          COALESCE(o.last_error, ''), o.created_at,
          (SELECT c.risk_level FROM commands c WHERE c.id = o.aggregate_id)`
	interval := fmt.Sprintf("%f seconds", leaseDuration.Seconds())
	rows, err := r.db.QueryContext(ctx, query, workerID, limit, now, interval)
	if err != nil {
		return nil, fmt.Errorf("lease outbox rows: %w", err)
	}
	defer rows.Close()
	var deliveries []Delivery
	for rows.Next() {
		var delivery Delivery
		var payload []byte
		var lockedAt, publishedAt sql.NullTime
		if err := rows.Scan(
			&delivery.Event.ID, &delivery.Event.WorkspaceID, &delivery.Event.AggregateType,
			&delivery.Event.AggregateID, &delivery.Event.EventType, &delivery.Event.Destination,
			&payload, &delivery.Event.Status, &delivery.Event.AttemptCount, &delivery.Event.AvailableAt,
			&lockedAt, &delivery.Event.LockedBy, &publishedAt, &delivery.Event.LastError,
			&delivery.Event.CreatedAt, &delivery.RiskLevel,
		); err != nil {
			return nil, err
		}
		delivery.Event.Payload = json.RawMessage(payload)
		delivery.Event.LockedAt = nullTimePointer(lockedAt)
		delivery.Event.PublishedAt = nullTimePointer(publishedAt)
		delivery.QoS = 1
		deliveries = append(deliveries, delivery)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return deliveries, nil
}

func (r *PostgresRepository) MarkPublished(ctx context.Context, workerID string, outboxID, commandID domain.ID, at time.Time) (domain.Command, bool, error) {
	return r.completeOutbox(ctx, workerID, outboxID, commandID, at, "", true)
}

func (r *PostgresRepository) MarkRetry(ctx context.Context, workerID string, outboxID domain.ID, availableAt time.Time, reason string) error {
	const query = `
UPDATE outbox_events
SET status = 'RETRY', available_at = $2, locked_at = NULL, locked_by = NULL, last_error = $3
WHERE id = $1 AND status = 'PROCESSING' AND locked_by = $4`
	result, err := r.db.ExecContext(ctx, query, outboxID, availableAt, reason, workerID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		var status string
		if queryErr := r.db.QueryRowContext(ctx, `SELECT status FROM outbox_events WHERE id = $1`, outboxID).Scan(&status); queryErr != nil {
			return queryErr
		}
		if status == "PUBLISHED" || status == "FAILED" {
			return nil
		}
		return errors.New("outbox lease is not owned by worker")
	}
	return nil
}

func (r *PostgresRepository) MarkFailed(ctx context.Context, workerID string, outboxID, commandID domain.ID, at time.Time, reason string) (domain.Command, bool, error) {
	return r.completeOutbox(ctx, workerID, outboxID, commandID, at, reason, false)
}

func (r *PostgresRepository) completeOutbox(ctx context.Context, workerID string, outboxID, commandID domain.ID, at time.Time, reason string, published bool) (domain.Command, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Command{}, false, err
	}
	defer tx.Rollback()
	status := "FAILED"
	if published {
		status = "PUBLISHED"
	}
	const updateOutbox = `
UPDATE outbox_events
SET status = $3, published_at = CASE WHEN $3 = 'PUBLISHED' THEN $4 ELSE published_at END,
    locked_at = NULL, locked_by = NULL, last_error = NULLIF($5, '')
WHERE id = $1 AND aggregate_id = $2 AND status = 'PROCESSING' AND locked_by = $6`
	result, err := tx.ExecContext(ctx, updateOutbox, outboxID, commandID, status, at, reason, workerID)
	if err != nil {
		return domain.Command{}, false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return domain.Command{}, false, err
	}
	if rows != 1 {
		var existingStatus string
		if queryErr := tx.QueryRowContext(ctx,
			`SELECT status FROM outbox_events WHERE id = $1 AND aggregate_id = $2`,
			outboxID, commandID,
		).Scan(&existingStatus); queryErr != nil {
			return domain.Command{}, false, queryErr
		}
		if (published && existingStatus == "PUBLISHED") || (!published && existingStatus == "FAILED") {
			command, queryErr := queryCommand(ctx, tx, `SELECT `+commandColumns+` FROM commands WHERE id = $1`, commandID)
			if queryErr != nil {
				return domain.Command{}, false, queryErr
			}
			if commitErr := tx.Commit(); commitErr != nil {
				return domain.Command{}, false, commitErr
			}
			return command, false, nil
		}
		return domain.Command{}, false, errors.New("outbox lease is not owned by worker")
	}
	command, err := queryCommand(ctx, tx, `SELECT `+commandColumns+` FROM commands WHERE id = $1 FOR UPDATE`, commandID)
	if err != nil {
		return domain.Command{}, false, err
	}
	if command.Status != domain.CommandPublishPending {
		if err := tx.Commit(); err != nil {
			return domain.Command{}, false, err
		}
		return command, false, nil
	}
	target := domain.CommandFailed
	action := "command.publish_failed"
	message := reason
	if published {
		target = domain.CommandPublished
		action = "command.published"
		message = "MQTT publish acknowledged"
	}
	event, err := command.Transition(target, at, "outbox", message)
	if err != nil {
		return domain.Command{}, false, err
	}
	if err := updateCommand(ctx, tx, command); err != nil {
		return domain.Command{}, false, err
	}
	if err := insertCommandEvent(ctx, tx, event, ""); err != nil {
		return domain.Command{}, false, err
	}
	if err := insertAudit(ctx, tx, commandAudit(command, action, "system", at)); err != nil {
		return domain.Command{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Command{}, false, err
	}
	return command, true, nil
}

func (r *PostgresRepository) ApplyReply(ctx context.Context, reply Reply) (domain.Command, bool, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Command{}, false, false, err
	}
	defer tx.Rollback()
	command, err := queryCommand(ctx, tx, `
SELECT `+commandColumns+`
FROM commands
WHERE workspace_id = $1 AND dji_tid = $2 AND dji_bid = $3 AND method = $4
FOR UPDATE`, reply.WorkspaceID, reply.TID, reply.BID, reply.Method)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Command{}, false, false, nil
	}
	if err != nil {
		return domain.Command{}, false, false, err
	}
	var pendingEvent *domain.CommandEvent
	if command.Status == domain.CommandPublishPending {
		event, transitionErr := command.Transition(domain.CommandPublished, reply.ReceivedAt, "dji", "reply observed before publisher completion")
		if transitionErr != nil {
			return domain.Command{}, true, false, transitionErr
		}
		pendingEvent = &event
	}
	if isTerminal(command.Status) || (command.Status == domain.CommandExecuting && reply.Status == domain.CommandAccepted) {
		if err := tx.Commit(); err != nil {
			return domain.Command{}, true, false, err
		}
		return command, true, false, nil
	}
	event, err := command.Transition(reply.Status, reply.ReceivedAt, "dji", reply.Message)
	if err != nil {
		return domain.Command{}, true, false, err
	}
	if event.ToStatus == "" {
		if err := tx.Commit(); err != nil {
			return domain.Command{}, true, false, err
		}
		return command, true, false, nil
	}
	event.ResultCode = reply.ResultCode
	command.ResultCode = reply.ResultCode
	command.ResultMessage = reply.Message
	if err := updateCommand(ctx, tx, command); err != nil {
		return domain.Command{}, true, false, err
	}
	if pendingEvent != nil {
		if err := insertCommandEvent(ctx, tx, *pendingEvent, reply.RawMessageID); err != nil {
			return domain.Command{}, true, false, err
		}
	}
	if err := insertCommandEvent(ctx, tx, event, reply.RawMessageID); err != nil {
		return domain.Command{}, true, false, err
	}
	if err := insertAudit(ctx, tx, commandAudit(command, "command.reply", "dji", reply.ReceivedAt)); err != nil {
		return domain.Command{}, true, false, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Command{}, true, false, err
	}
	return command, true, true, nil
}

func (r *PostgresRepository) RecordOrphanReply(ctx context.Context, reply OrphanReply) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	const insert = `
INSERT INTO orphan_command_replies (
  workspace_id, tid, bid, method, gateway_sn, payload_hash, payload, received_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (workspace_id, tid, bid, method, payload_hash) DO NOTHING`
	result, err := tx.ExecContext(ctx, insert,
		reply.WorkspaceID, reply.TID, reply.BID, reply.Method, reply.GatewaySN,
		reply.PayloadHash, jsonOrObject(reply.Payload), reply.ReceivedAt,
	)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if rows == 1 {
		if err := insertAudit(ctx, tx, AuditRecord{
			WorkspaceID: reply.WorkspaceID, ActorID: "dji", Action: "command.orphan_reply",
			ResourceType: "command_reply", Details: jsonOrObject(reply.Payload), CreatedAt: reply.ReceivedAt,
		}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return rows == 1, nil
}

func (r *PostgresRepository) Expire(ctx context.Context, now time.Time, limit int) ([]domain.Command, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `
SELECT `+commandColumns+`
FROM commands
WHERE status IN ('PUBLISHED', 'ACCEPTED', 'EXECUTING') AND expires_at <= $1
ORDER BY expires_at, id
FOR UPDATE SKIP LOCKED
LIMIT $2`, now, limit)
	if err != nil {
		return nil, err
	}
	var commands []domain.Command
	for rows.Next() {
		command, scanErr := scanCommand(rows)
		if scanErr != nil {
			rows.Close()
			return nil, scanErr
		}
		commands = append(commands, command)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range commands {
		event, transitionErr := commands[index].Transition(domain.CommandTimeout, now, "timeout", "command deadline exceeded")
		if transitionErr != nil {
			return nil, transitionErr
		}
		if err := updateCommand(ctx, tx, commands[index]); err != nil {
			return nil, err
		}
		if err := insertCommandEvent(ctx, tx, event, ""); err != nil {
			return nil, err
		}
		if err := insertAudit(ctx, tx, commandAudit(commands[index], "command.timeout", "system", now)); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return commands, nil
}

const commandColumns = `id, workspace_id, target_device_id, gateway_device_id, method, status,
risk_level, idempotency_key, request_hash, dji_tid, dji_bid, parameters, requested_by,
result_code, COALESCE(result_message, ''), created_at, expires_at, completed_at, updated_at`

type queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func queryCommand(ctx context.Context, queryer queryer, query string, args ...any) (domain.Command, error) {
	return scanCommand(queryer.QueryRowContext(ctx, query, args...))
}

type scanner interface {
	Scan(...any) error
}

func scanCommand(row scanner) (domain.Command, error) {
	var command domain.Command
	var gatewayID, tid, bid sql.NullString
	var resultCode sql.NullInt64
	var completedAt sql.NullTime
	var requestHash, parameters []byte
	err := row.Scan(
		&command.ID, &command.WorkspaceID, &command.TargetDeviceID, &gatewayID,
		&command.Method, &command.Status, &command.RiskLevel, &command.IdempotencyKey,
		&requestHash, &tid, &bid, &parameters, &command.RequestedBy, &resultCode,
		&command.ResultMessage, &command.CreatedAt, &command.ExpiresAt, &completedAt, &command.UpdatedAt,
	)
	if err != nil {
		return domain.Command{}, err
	}
	if gatewayID.Valid {
		command.GatewayDeviceID = domain.ID(gatewayID.String)
	}
	if tid.Valid {
		command.DJITID = tid.String
	}
	if bid.Valid {
		command.DJIBID = bid.String
	}
	if resultCode.Valid {
		value := int(resultCode.Int64)
		command.ResultCode = &value
	}
	command.CompletedAt = nullTimePointer(completedAt)
	command.RequestHash = hex.EncodeToString(requestHash)
	command.Parameters = json.RawMessage(parameters)
	return command, nil
}

func updateCommand(ctx context.Context, tx *sql.Tx, command domain.Command) error {
	const query = `
UPDATE commands
SET status = $2, result_code = $3, result_message = NULLIF($4, ''),
    completed_at = $5, updated_at = $6
WHERE id = $1`
	_, err := tx.ExecContext(ctx, query,
		command.ID, command.Status, command.ResultCode, command.ResultMessage,
		command.CompletedAt, command.UpdatedAt,
	)
	return err
}

func insertCommandEvent(ctx context.Context, tx *sql.Tx, event domain.CommandEvent, rawMessageID domain.ID) error {
	if event.ID == "" {
		id, err := newUUID()
		if err != nil {
			return err
		}
		event.ID = id
	}
	const query = `
INSERT INTO command_events (
  id, command_id, from_status, to_status, source, result_code, message, raw_message_id, occurred_at
) VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, NULLIF($7, ''), NULLIF($8, '')::uuid, $9)`
	_, err := tx.ExecContext(ctx, query,
		event.ID, event.CommandID, event.FromStatus, event.ToStatus, event.Source,
		event.ResultCode, event.Message, rawMessageID, event.OccurredAt,
	)
	return err
}

func insertAudit(ctx context.Context, tx *sql.Tx, audit AuditRecord) error {
	const query = `
INSERT INTO audit_logs (
  workspace_id, actor_id, action, resource_type, resource_id, request_id, details, created_at
) VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7, $8)`
	_, err := tx.ExecContext(ctx, query,
		audit.WorkspaceID, audit.ActorID, audit.Action, audit.ResourceType,
		audit.ResourceID, audit.RequestID, jsonOrObject(audit.Details), audit.CreatedAt,
	)
	return err
}

func jsonOrObject(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	return raw
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	result := value.Time
	return &result
}
