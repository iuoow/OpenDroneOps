package command

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type MemoryRepository struct {
	mu          sync.Mutex
	commands    map[domain.ID]domain.Command
	idempotency map[string]domain.ID
	outbox      map[domain.ID]domain.OutboxEvent
	events      []domain.CommandEvent
	audits      []AuditRecord
	orphans     map[string]OrphanReply
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		commands: make(map[domain.ID]domain.Command), idempotency: make(map[string]domain.ID),
		outbox: make(map[domain.ID]domain.OutboxEvent), orphans: make(map[string]OrphanReply),
	}
}

func (r *MemoryRepository) Create(_ context.Context, bundle CreateBundle) (domain.Command, bool, error) {
	key := string(bundle.Command.WorkspaceID) + "\x00" + bundle.Command.IdempotencyKey
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.idempotency[key]; ok {
		existing := r.commands[id]
		if err := domain.EnsureIdempotency(existing.RequestHash, bundle.Command.RequestHash); err != nil {
			return domain.Command{}, false, err
		}
		return cloneCommand(existing), false, nil
	}
	r.commands[bundle.Command.ID] = cloneCommand(bundle.Command)
	r.idempotency[key] = bundle.Command.ID
	r.outbox[bundle.Outbox.ID] = cloneOutbox(bundle.Outbox)
	r.events = append(r.events, bundle.Events...)
	r.audits = append(r.audits, bundle.Audit)
	return cloneCommand(bundle.Command), true, nil
}

func (r *MemoryRepository) LeaseOutbox(_ context.Context, workerID string, limit int, now time.Time, leaseDuration time.Duration) ([]Delivery, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	events := make([]domain.OutboxEvent, 0)
	for _, event := range r.outbox {
		leaseExpired := event.LockedAt != nil && !event.LockedAt.Add(leaseDuration).After(now)
		available := (event.Status == "PENDING" || event.Status == "RETRY") && !event.AvailableAt.After(now)
		if !available && !(event.Status == "PROCESSING" && leaseExpired) {
			continue
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].AvailableAt.Equal(events[j].AvailableAt) {
			return events[i].ID < events[j].ID
		}
		return events[i].AvailableAt.Before(events[j].AvailableAt)
	})
	if len(events) > limit {
		events = events[:limit]
	}
	deliveries := make([]Delivery, 0, len(events))
	for _, event := range events {
		lockedAt := now
		event.Status = "PROCESSING"
		event.LockedAt = &lockedAt
		event.LockedBy = workerID
		event.AttemptCount++
		r.outbox[event.ID] = event
		command := r.commands[event.AggregateID]
		deliveries = append(deliveries, Delivery{Event: cloneOutbox(event), RiskLevel: command.RiskLevel, QoS: 1})
	}
	return deliveries, nil
}

func (r *MemoryRepository) MarkPublished(_ context.Context, workerID string, outboxID, commandID domain.ID, at time.Time) (domain.Command, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event, ok := r.outbox[outboxID]
	if !ok || event.AggregateID != commandID {
		return domain.Command{}, false, errors.New("outbox event not found")
	}
	command, ok := r.commands[commandID]
	if !ok {
		return domain.Command{}, false, errors.New("command not found")
	}
	if event.Status == "PUBLISHED" {
		return cloneCommand(command), false, nil
	}
	if event.Status != "PROCESSING" || event.LockedBy != workerID {
		return domain.Command{}, false, errors.New("outbox lease is not owned by worker")
	}
	event.Status = "PUBLISHED"
	event.PublishedAt = &at
	event.LockedAt = nil
	event.LockedBy = ""
	event.LastError = ""
	r.outbox[outboxID] = event
	if command.Status != domain.CommandPublishPending {
		return cloneCommand(command), false, nil
	}
	transition, err := command.Transition(domain.CommandPublished, at, "outbox", "MQTT publish acknowledged")
	if err != nil {
		return domain.Command{}, false, err
	}
	r.commands[commandID] = command
	r.events = append(r.events, transition)
	r.audits = append(r.audits, commandAudit(command, "command.published", "system", at))
	return cloneCommand(command), true, nil
}

func (r *MemoryRepository) MarkRetry(_ context.Context, workerID string, outboxID domain.ID, availableAt time.Time, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	event, ok := r.outbox[outboxID]
	if !ok {
		return errors.New("outbox event not found")
	}
	if event.Status == "PUBLISHED" || event.Status == "FAILED" {
		return nil
	}
	if event.Status != "PROCESSING" || event.LockedBy != workerID {
		return errors.New("outbox lease is not owned by worker")
	}
	event.Status = "RETRY"
	event.AvailableAt = availableAt
	event.LockedAt = nil
	event.LockedBy = ""
	event.LastError = reason
	r.outbox[outboxID] = event
	return nil
}

func (r *MemoryRepository) MarkFailed(_ context.Context, workerID string, outboxID, commandID domain.ID, at time.Time, reason string) (domain.Command, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	event, ok := r.outbox[outboxID]
	if !ok || event.AggregateID != commandID {
		return domain.Command{}, false, errors.New("outbox event not found")
	}
	command, ok := r.commands[commandID]
	if !ok {
		return domain.Command{}, false, errors.New("command not found")
	}
	if event.Status != "PROCESSING" || event.LockedBy != workerID {
		return domain.Command{}, false, errors.New("outbox lease is not owned by worker")
	}
	event.Status = "FAILED"
	event.LockedAt = nil
	event.LockedBy = ""
	event.LastError = reason
	r.outbox[outboxID] = event
	if command.Status != domain.CommandPublishPending {
		return cloneCommand(command), false, nil
	}
	transition, err := command.Transition(domain.CommandFailed, at, "outbox", reason)
	if err != nil {
		return domain.Command{}, false, err
	}
	command.ResultMessage = reason
	r.commands[commandID] = command
	r.events = append(r.events, transition)
	r.audits = append(r.audits, commandAudit(command, "command.publish_failed", "system", at))
	return cloneCommand(command), true, nil
}

func (r *MemoryRepository) ApplyReply(_ context.Context, reply Reply) (domain.Command, bool, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var command domain.Command
	var found bool
	for _, candidate := range r.commands {
		if candidate.WorkspaceID == reply.WorkspaceID && candidate.DJITID == reply.TID &&
			candidate.DJIBID == reply.BID && candidate.Method == reply.Method {
			command, found = candidate, true
			break
		}
	}
	if !found {
		return domain.Command{}, false, false, nil
	}
	if command.Status == domain.CommandPublishPending {
		event, err := command.Transition(domain.CommandPublished, reply.ReceivedAt, "dji", "reply observed before publisher completion")
		if err != nil {
			return domain.Command{}, true, false, err
		}
		r.events = append(r.events, event)
	}
	if isTerminal(command.Status) {
		return cloneCommand(command), true, false, nil
	}
	if command.Status == domain.CommandExecuting && reply.Status == domain.CommandAccepted {
		return cloneCommand(command), true, false, nil
	}
	event, err := command.Transition(reply.Status, reply.ReceivedAt, "dji", reply.Message)
	if err != nil {
		return domain.Command{}, true, false, err
	}
	if event.ToStatus == "" {
		return cloneCommand(command), true, false, nil
	}
	event.ResultCode = reply.ResultCode
	command.ResultCode = reply.ResultCode
	command.ResultMessage = reply.Message
	r.commands[command.ID] = command
	r.events = append(r.events, event)
	r.audits = append(r.audits, commandAudit(command, "command.reply", "dji", reply.ReceivedAt))
	return cloneCommand(command), true, true, nil
}

func (r *MemoryRepository) RecordOrphanReply(_ context.Context, reply OrphanReply) (bool, error) {
	key := string(reply.WorkspaceID) + "\x00" + reply.TID + "\x00" + reply.BID + "\x00" + reply.Method + "\x00" + reply.PayloadHash
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.orphans[key]; exists {
		return false, nil
	}
	r.orphans[key] = reply
	r.audits = append(r.audits, AuditRecord{
		WorkspaceID: reply.WorkspaceID, ActorID: "dji", Action: "command.orphan_reply",
		ResourceType: "command_reply", Details: reply.Payload, CreatedAt: reply.ReceivedAt,
	})
	return true, nil
}

func (r *MemoryRepository) Expire(_ context.Context, now time.Time, limit int) ([]domain.Command, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := make([]domain.ID, 0)
	for id, command := range r.commands {
		if command.ExpiresAt.After(now) || !canTimeout(command.Status) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > limit {
		ids = ids[:limit]
	}
	expired := make([]domain.Command, 0, len(ids))
	for _, id := range ids {
		command := r.commands[id]
		event, err := command.Transition(domain.CommandTimeout, now, "timeout", "command deadline exceeded")
		if err != nil {
			return nil, err
		}
		r.commands[id] = command
		r.events = append(r.events, event)
		r.audits = append(r.audits, commandAudit(command, "command.timeout", "system", now))
		expired = append(expired, cloneCommand(command))
	}
	return expired, nil
}

func commandAudit(command domain.Command, action, actor string, at time.Time) AuditRecord {
	return AuditRecord{
		WorkspaceID: command.WorkspaceID, ActorID: actor, Action: action,
		ResourceType: "command", ResourceID: command.ID, CreatedAt: at,
	}
}

func canTimeout(status domain.CommandStatus) bool {
	return status == domain.CommandPublished || status == domain.CommandAccepted || status == domain.CommandExecuting
}

func isTerminal(status domain.CommandStatus) bool {
	switch status {
	case domain.CommandSucceeded, domain.CommandFailed, domain.CommandTimeout, domain.CommandCanceled, domain.CommandRejected:
		return true
	default:
		return false
	}
}

func cloneCommand(command domain.Command) domain.Command {
	command.Parameters = append([]byte(nil), command.Parameters...)
	return command
}

func cloneOutbox(event domain.OutboxEvent) domain.OutboxEvent {
	event.Payload = append([]byte(nil), event.Payload...)
	return event
}
