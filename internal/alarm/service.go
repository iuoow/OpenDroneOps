package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

var (
	ErrInvalidFinding = errors.New("invalid alarm finding")
	ErrAlarmNotFound  = errors.New("alarm not found")
)

type Finding struct {
	WorkspaceID domain.ID
	DeviceID    domain.ID
	DedupKey    string
	AlarmType   string
	Severity    domain.AlarmSeverity
	OccurredAt  time.Time
	Details     json.RawMessage
}

type Resolution struct {
	WorkspaceID domain.ID
	DeviceID    domain.ID
	DedupKey    string
	OccurredAt  time.Time
}

type Evaluation struct {
	Findings    []Finding
	Resolutions []Resolution
}

type Input struct {
	State      *domain.DeviceState
	Event      *domain.DeviceEvent
	ObservedAt time.Time
}

type Rule interface {
	Name() string
	Evaluate(Input) (Evaluation, error)
}

type AlarmStore interface {
	UpsertFinding(context.Context, Finding) (domain.Alarm, bool, error)
	Acknowledge(context.Context, domain.ID, domain.ID, string, time.Time) (domain.Alarm, bool, error)
	Resolve(context.Context, domain.ID, domain.ID, time.Time) (domain.Alarm, bool, error)
	ResolveByDedup(context.Context, Resolution) (domain.Alarm, bool, error)
	ListActive(context.Context, domain.ID) ([]domain.Alarm, error)
}

type Publisher interface {
	Publish(websockethub.Event)
}

type Change struct {
	Type       string
	Alarm      domain.Alarm
	OccurredAt time.Time
}

type Service struct {
	store     AlarmStore
	rules     []Rule
	publisher Publisher
	now       func() time.Time
}

func NewService(store AlarmStore, rules []Rule, publisher Publisher) (*Service, error) {
	if store == nil {
		return nil, errors.New("alarm store is required")
	}
	if len(rules) == 0 {
		rules = DefaultRules()
	}
	return &Service{store: store, rules: append([]Rule(nil), rules...), publisher: publisher, now: func() time.Time {
		return time.Now().UTC()
	}}, nil
}

func DefaultRules() []Rule {
	return []Rule{
		OfflineRule{},
		LowBatteryRule{WarningBelow: 20, CriticalBelow: 10},
		ProtocolAlarmRule{},
	}
}

func (s *Service) Evaluate(ctx context.Context, input Input) ([]Change, error) {
	now := input.ObservedAt
	if now.IsZero() {
		now = s.now()
	}
	var changes []Change
	for _, rule := range s.rules {
		if rule == nil {
			continue
		}
		evaluation, err := rule.Evaluate(input)
		if err != nil {
			return nil, fmt.Errorf("evaluate alarm rule %s: %w", rule.Name(), err)
		}
		for _, finding := range evaluation.Findings {
			if finding.OccurredAt.IsZero() {
				finding.OccurredAt = now
			}
			if finding.WorkspaceID == "" && input.State != nil {
				finding.WorkspaceID = input.State.WorkspaceID
			}
			if finding.DeviceID == "" && input.State != nil {
				finding.DeviceID = input.State.DeviceID
			}
			if finding.WorkspaceID == "" && input.Event != nil {
				finding.WorkspaceID = input.Event.WorkspaceID
			}
			if finding.DeviceID == "" && input.Event != nil {
				finding.DeviceID = input.Event.DeviceID
			}
			alarm, created, err := s.store.UpsertFinding(ctx, finding)
			if err != nil {
				return nil, fmt.Errorf("persist alarm finding: %w", err)
			}
			changeType := "alarm.updated"
			if created {
				changeType = "alarm.created"
			}
			change := Change{Type: changeType, Alarm: alarm, OccurredAt: alarm.LastOccurredAt}
			changes = append(changes, change)
			s.publish(change)
		}
		for _, resolution := range evaluation.Resolutions {
			if resolution.OccurredAt.IsZero() {
				resolution.OccurredAt = now
			}
			if resolution.WorkspaceID == "" && input.State != nil {
				resolution.WorkspaceID = input.State.WorkspaceID
			}
			if resolution.DeviceID == "" && input.State != nil {
				resolution.DeviceID = input.State.DeviceID
			}
			if resolution.WorkspaceID == "" && input.Event != nil {
				resolution.WorkspaceID = input.Event.WorkspaceID
			}
			if resolution.DeviceID == "" && input.Event != nil {
				resolution.DeviceID = input.Event.DeviceID
			}
			alarm, changed, err := s.store.ResolveByDedup(ctx, resolution)
			if err != nil {
				return nil, fmt.Errorf("resolve alarm condition: %w", err)
			}
			if !changed {
				continue
			}
			change := Change{Type: "alarm.resolved", Alarm: alarm, OccurredAt: resolution.OccurredAt}
			changes = append(changes, change)
			s.publish(change)
		}
	}
	return changes, nil
}

func (s *Service) Acknowledge(ctx context.Context, workspaceID, alarmID domain.ID, actor string, at time.Time) (domain.Alarm, error) {
	if at.IsZero() {
		at = s.now()
	}
	alarm, changed, err := s.store.Acknowledge(ctx, workspaceID, alarmID, actor, at)
	if err != nil {
		return domain.Alarm{}, fmt.Errorf("acknowledge alarm: %w", err)
	}
	if changed {
		s.publish(Change{Type: "alarm.updated", Alarm: alarm, OccurredAt: at})
	}
	return alarm, nil
}

func (s *Service) Resolve(ctx context.Context, workspaceID, alarmID domain.ID, at time.Time) (domain.Alarm, error) {
	if at.IsZero() {
		at = s.now()
	}
	alarm, changed, err := s.store.Resolve(ctx, workspaceID, alarmID, at)
	if err != nil {
		return domain.Alarm{}, fmt.Errorf("resolve alarm: %w", err)
	}
	if changed {
		s.publish(Change{Type: "alarm.resolved", Alarm: alarm, OccurredAt: at})
	}
	return alarm, nil
}

func (s *Service) Recover(ctx context.Context, workspaceID domain.ID) ([]domain.Alarm, error) {
	alarms, err := s.store.ListActive(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("recover active alarms: %w", err)
	}
	return alarms, nil
}

func (s *Service) publish(change Change) {
	if s.publisher == nil {
		return
	}
	data, err := json.Marshal(change.Alarm)
	if err != nil {
		return
	}
	occurredAt := changeOccurredAt(change)
	s.publisher.Publish(websockethub.Event{
		EventID:       fmt.Sprintf("alarm:%s:%s:%d:%d", change.Alarm.ID, change.Type, change.Alarm.OccurrenceCount, occurredAt.UnixNano()),
		Type:          change.Type,
		SchemaVersion: "1.0",
		WorkspaceID:   string(change.Alarm.WorkspaceID),
		AggregateID:   string(change.Alarm.DeviceID),
		OccurredAt:    occurredAt,
		Data:          data,
	})
}

func validateFinding(finding Finding) error {
	if finding.WorkspaceID == "" || finding.DeviceID == "" || strings.TrimSpace(finding.DedupKey) == "" ||
		strings.TrimSpace(finding.AlarmType) == "" || finding.OccurredAt.IsZero() {
		return ErrInvalidFinding
	}
	if len(finding.Details) > 0 && !json.Valid(finding.Details) {
		return fmt.Errorf("%w: details must be valid JSON", ErrInvalidFinding)
	}
	switch finding.Severity {
	case domain.AlarmInfo, domain.AlarmWarning, domain.AlarmCritical:
		return nil
	default:
		return fmt.Errorf("%w: unsupported severity %q", ErrInvalidFinding, finding.Severity)
	}
}

func changeOccurredAt(change Change) time.Time {
	if !change.OccurredAt.IsZero() {
		return change.OccurredAt
	}
	return change.Alarm.LastOccurredAt
}
