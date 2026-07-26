package alarm

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

func TestEvaluateDeduplicatesActiveAlarmAndPublishesLifecycle(t *testing.T) {
	store := NewMemoryStore()
	publisher := &recordingPublisher{}
	service, err := NewService(store, []Rule{LowBatteryRule{WarningBelow: 20, CriticalBelow: 10}}, publisher)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	observedAt := time.Date(2026, 7, 26, 1, 2, 3, 0, time.UTC)
	battery := 8.0
	input := Input{State: &domain.DeviceState{
		WorkspaceID: "workspace-1", DeviceID: "device-1", BatteryPercent: &battery,
	}, ObservedAt: observedAt}
	changes, err := service.Evaluate(context.Background(), input)
	if err != nil || len(changes) != 1 || changes[0].Type != "alarm.created" {
		t.Fatalf("first evaluation changes=%+v err=%v", changes, err)
	}
	changes, err = service.Evaluate(context.Background(), input)
	if err != nil || len(changes) != 1 || changes[0].Type != "alarm.updated" {
		t.Fatalf("duplicate evaluation changes=%+v err=%v", changes, err)
	}
	active, err := service.Recover(context.Background(), "workspace-1")
	if err != nil || len(active) != 1 {
		t.Fatalf("active alarms=%+v err=%v", active, err)
	}
	if active[0].OccurrenceCount != 2 || active[0].Severity != domain.AlarmCritical {
		t.Fatalf("deduplication did not update occurrence: %+v", active[0])
	}
	if got := publisher.types(); len(got) != 2 || got[0] != "alarm.created" || got[1] != "alarm.updated" {
		t.Fatalf("published lifecycle=%v", got)
	}
	events := publisher.snapshot()
	if events[0].EventID == events[1].EventID || !json.Valid(events[0].Data) || events[0].WorkspaceID != "workspace-1" {
		t.Fatalf("invalid WebSocket alarm envelopes=%+v", events)
	}
}

func TestAcknowledgeResolveAndReopenUseStableLifecycle(t *testing.T) {
	store := NewMemoryStore()
	service, _ := NewService(store, []Rule{OfflineRule{}}, nil)
	at := time.Date(2026, 7, 26, 2, 0, 0, 0, time.UTC)
	offline := false
	input := Input{State: &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: "device-1", Online: offline}, ObservedAt: at}
	changes, err := service.Evaluate(context.Background(), input)
	if err != nil || len(changes) != 1 {
		t.Fatalf("offline evaluation changes=%+v err=%v", changes, err)
	}
	alarmID := changes[0].Alarm.ID
	alarm, err := service.Acknowledge(context.Background(), "workspace-1", alarmID, "operator-1", at.Add(time.Minute))
	if err != nil || alarm.Status != domain.AlarmAcknowledged {
		t.Fatalf("acknowledge alarm=%+v err=%v", alarm, err)
	}
	alarm, err = service.Acknowledge(context.Background(), "workspace-1", alarmID, "operator-1", at.Add(2*time.Minute))
	if err != nil || alarm.Status != domain.AlarmAcknowledged {
		t.Fatalf("idempotent acknowledge alarm=%+v err=%v", alarm, err)
	}
	alarm, err = service.Resolve(context.Background(), "workspace-1", alarmID, at.Add(3*time.Minute))
	if err != nil || alarm.Status != domain.AlarmResolved {
		t.Fatalf("resolve alarm=%+v err=%v", alarm, err)
	}
	online := true
	changes, err = service.Evaluate(context.Background(), Input{
		State:      &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: "device-1", Online: online},
		ObservedAt: at.Add(4 * time.Minute),
	})
	if err != nil || len(changes) != 0 {
		t.Fatalf("online resolution changes=%+v err=%v", changes, err)
	}
	offline = false
	changes, err = service.Evaluate(context.Background(), Input{
		State:      &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: "device-1", Online: offline},
		ObservedAt: at.Add(5 * time.Minute),
	})
	if err != nil || len(changes) != 1 || changes[0].Type != "alarm.created" || changes[0].Alarm.ID == alarmID {
		t.Fatalf("reopened alarm changes=%+v err=%v", changes, err)
	}
}

func TestClearedStateConditionAutomaticallyResolvesActiveAlarm(t *testing.T) {
	store := NewMemoryStore()
	service, _ := NewService(store, []Rule{LowBatteryRule{WarningBelow: 20, CriticalBelow: 10}}, nil)
	at := time.Now().UTC()
	low := 15.0
	changes, err := service.Evaluate(context.Background(), Input{
		State:      &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: "device-1", BatteryPercent: &low},
		ObservedAt: at,
	})
	if err != nil || len(changes) != 1 || changes[0].Type != "alarm.created" {
		t.Fatalf("low battery changes=%+v err=%v", changes, err)
	}
	healthy := 80.0
	changes, err = service.Evaluate(context.Background(), Input{
		State:      &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: "device-1", BatteryPercent: &healthy},
		ObservedAt: at.Add(time.Minute),
	})
	if err != nil || len(changes) != 1 || changes[0].Type != "alarm.resolved" {
		t.Fatalf("healthy battery changes=%+v err=%v", changes, err)
	}
	active, err := service.Recover(context.Background(), "workspace-1")
	if err != nil || len(active) != 0 {
		t.Fatalf("active alarms=%+v err=%v", active, err)
	}
}

func TestProtocolAlarmRuleUsesEventPayloadAndRejectsInvalidJSON(t *testing.T) {
	rule := ProtocolAlarmRule{}
	payload, _ := json.Marshal(map[string]any{
		"alarm_type": "propeller.blocked", "dedup_key": "propeller:blocked",
		"severity": "CRITICAL", "details": map[string]any{"motor": 2},
	})
	evaluation, err := rule.Evaluate(Input{Event: &domain.DeviceEvent{
		WorkspaceID: "workspace-1", DeviceID: "device-1", EventType: "ALARM",
		ReceivedAt: time.Now().UTC(), Payload: payload,
	}})
	if err != nil || len(evaluation.Findings) != 1 {
		t.Fatalf("protocol evaluation=%+v err=%v", evaluation, err)
	}
	if evaluation.Findings[0].Severity != domain.AlarmCritical || evaluation.Findings[0].DedupKey != "device:device-1:propeller:blocked" {
		t.Fatalf("protocol finding=%+v", evaluation.Findings[0])
	}
	_, err = rule.Evaluate(Input{Event: &domain.DeviceEvent{
		WorkspaceID: "workspace-1", DeviceID: "device-1", EventType: "ALARM",
		ReceivedAt: time.Now().UTC(), Payload: []byte("{"),
	}})
	if err == nil {
		t.Fatal("invalid protocol payload should fail")
	}
}

func TestMemoryStoreConcurrentDedupKeepsOneActiveAlarm(t *testing.T) {
	store := NewMemoryStore()
	finding := Finding{
		WorkspaceID: "workspace-1", DeviceID: "device-1", DedupKey: "device.offline",
		AlarmType: "device.offline", Severity: domain.AlarmWarning, OccurredAt: time.Now().UTC(),
	}
	const workers = 16
	var wait sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, _, err := store.UpsertFinding(context.Background(), finding)
			if err != nil {
				errs <- err
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	active, err := store.ListActive(context.Background(), "workspace-1")
	if err != nil || len(active) != 1 {
		t.Fatalf("active alarms=%+v err=%v", active, err)
	}
	if active[0].OccurrenceCount != workers {
		t.Fatalf("occurrence count=%d want=%d", active[0].OccurrenceCount, workers)
	}
}

func TestDefaultRuleDedupKeysAreScopedPerDevice(t *testing.T) {
	store := NewMemoryStore()
	service, _ := NewService(store, []Rule{OfflineRule{}}, nil)
	at := time.Now().UTC()
	for _, deviceID := range []domain.ID{"device-1", "device-2"} {
		_, err := service.Evaluate(context.Background(), Input{
			State:      &domain.DeviceState{WorkspaceID: "workspace-1", DeviceID: deviceID, Online: false},
			ObservedAt: at,
		})
		if err != nil {
			t.Fatalf("evaluate %s: %v", deviceID, err)
		}
	}
	active, err := service.Recover(context.Background(), "workspace-1")
	if err != nil || len(active) != 2 {
		t.Fatalf("active alarms=%+v err=%v", active, err)
	}
	if active[0].DedupKey == active[1].DedupKey {
		t.Fatalf("device alarm dedup keys collided: %q", active[0].DedupKey)
	}
}

func TestServiceRejectsMissingStoreAndUnknownAlarm(t *testing.T) {
	if _, err := NewService(nil, nil, nil); err == nil {
		t.Fatal("missing store should fail")
	}
	store := NewMemoryStore()
	service, _ := NewService(store, nil, nil)
	_, err := service.Acknowledge(context.Background(), "workspace-1", "missing", "operator", time.Now())
	if !errors.Is(err, ErrAlarmNotFound) {
		t.Fatalf("unknown alarm error=%v", err)
	}
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []websockethub.Event
}

func (p *recordingPublisher) Publish(event websockethub.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}

func (p *recordingPublisher) types() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([]string, len(p.events))
	for i, event := range p.events {
		result[i] = event.Type
	}
	return result
}

func (p *recordingPublisher) snapshot() []websockethub.Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]websockethub.Event(nil), p.events...)
}
