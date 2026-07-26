package alarm

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type MemoryStore struct {
	mu     sync.RWMutex
	nextID atomic.Uint64
	alarms map[domain.ID]domain.Alarm
	active map[string]domain.ID
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{alarms: make(map[domain.ID]domain.Alarm), active: make(map[string]domain.ID)}
}

func (s *MemoryStore) UpsertFinding(_ context.Context, finding Finding) (domain.Alarm, bool, error) {
	if err := validateFinding(finding); err != nil {
		return domain.Alarm{}, false, err
	}
	key := activeKey(finding.WorkspaceID, finding.DedupKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.active[key]; ok {
		alarm := s.alarms[id]
		if finding.OccurredAt.After(alarm.LastOccurredAt) {
			alarm.LastOccurredAt = finding.OccurredAt
		}
		alarm.OccurrenceCount++
		alarm.Severity = maxSeverity(alarm.Severity, finding.Severity)
		if len(finding.Details) > 0 {
			alarm.Details = append([]byte(nil), finding.Details...)
		}
		s.alarms[id] = alarm
		return alarm, false, nil
	}
	id := domain.ID("alarm-" + itoa(s.nextID.Add(1)))
	alarm := domain.Alarm{
		ID: id, WorkspaceID: finding.WorkspaceID, DeviceID: finding.DeviceID,
		DedupKey: finding.DedupKey, AlarmType: finding.AlarmType, Severity: finding.Severity,
		Status: domain.AlarmOpen, FirstOccurredAt: finding.OccurredAt, LastOccurredAt: finding.OccurredAt,
		OccurrenceCount: 1, Details: append([]byte(nil), finding.Details...),
	}
	s.alarms[id] = alarm
	s.active[key] = id
	return alarm, true, nil
}

func (s *MemoryStore) Acknowledge(_ context.Context, workspaceID, alarmID domain.ID, actor string, at time.Time) (domain.Alarm, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	alarm, ok := s.alarms[alarmID]
	if !ok || alarm.WorkspaceID != workspaceID {
		return domain.Alarm{}, false, ErrAlarmNotFound
	}
	was := alarm.Status
	if err := alarm.Acknowledge(actor, at); err != nil {
		return domain.Alarm{}, false, err
	}
	s.alarms[alarmID] = alarm
	return alarm, was != alarm.Status, nil
}

func (s *MemoryStore) Resolve(_ context.Context, workspaceID, alarmID domain.ID, at time.Time) (domain.Alarm, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	alarm, ok := s.alarms[alarmID]
	if !ok || alarm.WorkspaceID != workspaceID {
		return domain.Alarm{}, false, ErrAlarmNotFound
	}
	was := alarm.Status
	if err := alarm.Resolve(at); err != nil {
		return domain.Alarm{}, false, err
	}
	s.alarms[alarmID] = alarm
	delete(s.active, activeKey(alarm.WorkspaceID, alarm.DedupKey))
	return alarm, was != alarm.Status, nil
}

func (s *MemoryStore) ResolveByDedup(_ context.Context, resolution Resolution) (domain.Alarm, bool, error) {
	if resolution.WorkspaceID == "" || resolution.DedupKey == "" {
		return domain.Alarm{}, false, nil
	}
	key := activeKey(resolution.WorkspaceID, resolution.DedupKey)
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.active[key]
	if !ok {
		return domain.Alarm{}, false, nil
	}
	alarm := s.alarms[id]
	if err := alarm.Resolve(resolution.OccurredAt); err != nil {
		return domain.Alarm{}, false, err
	}
	s.alarms[id] = alarm
	delete(s.active, key)
	return alarm, true, nil
}

func (s *MemoryStore) ListActive(_ context.Context, workspaceID domain.ID) ([]domain.Alarm, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	alarms := make([]domain.Alarm, 0)
	for _, alarm := range s.alarms {
		if alarm.WorkspaceID == workspaceID && (alarm.Status == domain.AlarmOpen || alarm.Status == domain.AlarmAcknowledged) {
			alarms = append(alarms, alarm)
		}
	}
	sort.Slice(alarms, func(i, j int) bool { return alarms[i].ID < alarms[j].ID })
	return alarms, nil
}

func activeKey(workspaceID domain.ID, dedupKey string) string {
	return string(workspaceID) + "\x00" + dedupKey
}

func maxSeverity(left, right domain.AlarmSeverity) domain.AlarmSeverity {
	if severityRank(right) > severityRank(left) {
		return right
	}
	return left
}

func severityRank(severity domain.AlarmSeverity) int {
	switch severity {
	case domain.AlarmCritical:
		return 3
	case domain.AlarmWarning:
		return 2
	case domain.AlarmInfo:
		return 1
	default:
		return 0
	}
}

func itoa(value uint64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buffer [20]byte
	index := len(buffer)
	for value > 0 {
		index--
		buffer[index] = digits[value%10]
		value /= 10
	}
	return string(buffer[index:])
}
