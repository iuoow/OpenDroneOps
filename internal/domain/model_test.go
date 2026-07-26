package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDeviceStateApplyRejectsStaleVersions(t *testing.T) {
	current := DeviceState{DeviceID: "device-1", WorkspaceID: "workspace-1", StateVersion: 4}
	if err := current.Apply(DeviceState{DeviceID: "device-1", WorkspaceID: "workspace-1", StateVersion: 3}); !errors.Is(err, ErrStaleState) {
		t.Fatalf("expected stale state error, got %v", err)
	}
	if current.StateVersion != 4 {
		t.Fatalf("stale update changed state version to %d", current.StateVersion)
	}
	if err := current.Apply(DeviceState{DeviceID: "device-1", WorkspaceID: "workspace-1", StateVersion: 5}); err != nil {
		t.Fatalf("newer state rejected: %v", err)
	}
}

func TestCommandTransitionRules(t *testing.T) {
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	command := Command{ID: "cmd-1", Status: CommandCreated}
	event, err := command.Transition(CommandValidated, now, "api", "")
	if err != nil || event.FromStatus != CommandCreated || command.Status != CommandValidated {
		t.Fatalf("unexpected transition: event=%+v err=%v command=%+v", event, err, command)
	}
	if _, err := command.Transition(CommandSucceeded, now, "api", ""); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
	if _, err := command.Transition(CommandPublishPending, now, "worker", ""); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}
	if _, err := command.Transition(CommandPublished, now, "worker", ""); err != nil {
		t.Fatalf("valid transition rejected: %v", err)
	}
	if _, err := command.Transition(CommandSucceeded, now, "dji", "ok"); err != nil {
		t.Fatalf("terminal transition rejected: %v", err)
	}
	if command.CompletedAt == nil {
		t.Fatal("terminal transition did not set completed_at")
	}
}

func TestIdempotencyConflict(t *testing.T) {
	if err := EnsureIdempotency("same", "same"); err != nil {
		t.Fatalf("same request hash rejected: %v", err)
	}
	if !errors.Is(EnsureIdempotency("a", "b"), ErrIdempotencyConflict) {
		t.Fatal("different request hashes should conflict")
	}
}

func TestAlarmLifecycle(t *testing.T) {
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	alarm := Alarm{Status: AlarmOpen}
	if err := alarm.Acknowledge("operator-1", now); err != nil {
		t.Fatalf("acknowledge failed: %v", err)
	}
	if err := alarm.Resolve(now); err != nil || alarm.Status != AlarmResolved {
		t.Fatalf("resolve failed: status=%s err=%v", alarm.Status, err)
	}
}
