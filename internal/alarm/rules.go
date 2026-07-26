package alarm

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

type OfflineRule struct{}

func (OfflineRule) Name() string { return "device-offline" }

func (OfflineRule) Evaluate(input Input) (Evaluation, error) {
	if input.State == nil {
		return Evaluation{}, nil
	}
	if input.State.Online {
		return Evaluation{Resolutions: []Resolution{{DedupKey: deviceDedup(input.State.DeviceID, "device.offline")}}}, nil
	}
	return Evaluation{Findings: []Finding{{
		WorkspaceID: input.State.WorkspaceID,
		DeviceID:    input.State.DeviceID,
		DedupKey:    deviceDedup(input.State.DeviceID, "device.offline"),
		AlarmType:   "device.offline",
		Severity:    domain.AlarmWarning,
		OccurredAt:  input.ObservedAt,
	}}}, nil
}

type LowBatteryRule struct {
	WarningBelow  float64
	CriticalBelow float64
}

func (r LowBatteryRule) Name() string { return "battery-threshold" }

func (r LowBatteryRule) Evaluate(input Input) (Evaluation, error) {
	if input.State == nil || input.State.BatteryPercent == nil {
		return Evaluation{}, nil
	}
	warning := r.WarningBelow
	if warning <= 0 {
		warning = 20
	}
	critical := r.CriticalBelow
	if critical <= 0 || critical >= warning {
		critical = warning / 2
	}
	value := *input.State.BatteryPercent
	if value > warning {
		return Evaluation{Resolutions: []Resolution{{
			DedupKey: deviceDedup(input.State.DeviceID, "battery.low"),
		}}}, nil
	}
	severity := domain.AlarmWarning
	if value <= critical {
		severity = domain.AlarmCritical
	}
	details, _ := json.Marshal(map[string]any{"battery_percent": value, "warning_below": warning, "critical_below": critical})
	return Evaluation{Findings: []Finding{{
		WorkspaceID: input.State.WorkspaceID,
		DeviceID:    input.State.DeviceID,
		DedupKey:    deviceDedup(input.State.DeviceID, "battery.low"),
		AlarmType:   "battery.low",
		Severity:    severity,
		OccurredAt:  input.ObservedAt,
		Details:     details,
	}}}, nil
}

type ProtocolAlarmRule struct {
	EventTypes map[string]struct{}
}

func (r ProtocolAlarmRule) Name() string { return "protocol-alarm" }

func (r ProtocolAlarmRule) Evaluate(input Input) (Evaluation, error) {
	if input.Event == nil {
		return Evaluation{}, nil
	}
	eventType := strings.ToLower(strings.TrimSpace(input.Event.EventType))
	if !r.matches(eventType) {
		return Evaluation{}, nil
	}
	var payload struct {
		AlarmType string               `json:"alarm_type"`
		DedupKey  string               `json:"dedup_key"`
		Severity  domain.AlarmSeverity `json:"severity"`
		Details   json.RawMessage      `json:"details"`
	}
	if err := json.Unmarshal(input.Event.Payload, &payload); err != nil {
		return Evaluation{}, fmt.Errorf("decode protocol alarm payload: %w", err)
	}
	if payload.AlarmType == "" {
		payload.AlarmType = "protocol.alarm"
	}
	if payload.DedupKey == "" {
		payload.DedupKey = payload.AlarmType
	}
	payload.DedupKey = deviceDedup(input.Event.DeviceID, payload.DedupKey)
	if payload.Severity == "" {
		payload.Severity = domain.AlarmWarning
	}
	if len(payload.Details) == 0 {
		payload.Details = input.Event.Payload
	}
	return Evaluation{Findings: []Finding{{
		WorkspaceID: input.Event.WorkspaceID,
		DeviceID:    input.Event.DeviceID,
		DedupKey:    payload.DedupKey,
		AlarmType:   payload.AlarmType,
		Severity:    payload.Severity,
		OccurredAt:  input.Event.ReceivedAt,
		Details:     payload.Details,
	}}}, nil
}

func deviceDedup(deviceID domain.ID, condition string) string {
	return "device:" + string(deviceID) + ":" + condition
}

func (r ProtocolAlarmRule) matches(eventType string) bool {
	if len(r.EventTypes) > 0 {
		_, ok := r.EventTypes[eventType]
		return ok
	}
	return eventType == "alarm" || eventType == "alarm.triggered" || eventType == "alarm.created"
}
