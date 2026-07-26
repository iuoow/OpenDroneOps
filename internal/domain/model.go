package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type ID string

type WorkspaceStatus string

const (
	WorkspaceActive WorkspaceStatus = "ACTIVE"
	WorkspacePaused WorkspaceStatus = "PAUSED"
)

type DeviceType string

const (
	DeviceTypeGateway  DeviceType = "GATEWAY"
	DeviceTypeAircraft DeviceType = "AIRCRAFT"
	DeviceTypeUnknown  DeviceType = "UNKNOWN"
)

type DeviceStatus string

const (
	DeviceRegistered DeviceStatus = "REGISTERED"
	DeviceOnline     DeviceStatus = "ONLINE"
	DeviceOffline    DeviceStatus = "OFFLINE"
)

type AlarmSeverity string

const (
	AlarmInfo     AlarmSeverity = "INFO"
	AlarmWarning  AlarmSeverity = "WARNING"
	AlarmCritical AlarmSeverity = "CRITICAL"
)

type AlarmStatus string

const (
	AlarmOpen         AlarmStatus = "OPEN"
	AlarmAcknowledged AlarmStatus = "ACKNOWLEDGED"
	AlarmResolved     AlarmStatus = "RESOLVED"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "LOW"
	RiskMedium RiskLevel = "MEDIUM"
	RiskHigh   RiskLevel = "HIGH"
)

type CommandStatus string

const (
	CommandCreated        CommandStatus = "CREATED"
	CommandValidated      CommandStatus = "VALIDATED"
	CommandRejected       CommandStatus = "REJECTED"
	CommandPublishPending CommandStatus = "PUBLISH_PENDING"
	CommandPublished      CommandStatus = "PUBLISHED"
	CommandAccepted       CommandStatus = "ACCEPTED"
	CommandExecuting      CommandStatus = "EXECUTING"
	CommandSucceeded      CommandStatus = "SUCCEEDED"
	CommandFailed         CommandStatus = "FAILED"
	CommandTimeout        CommandStatus = "TIMEOUT"
	CommandCanceled       CommandStatus = "CANCELED"
)

var (
	ErrStaleState          = errors.New("incoming device state is stale")
	ErrInvalidTransition   = errors.New("invalid command status transition")
	ErrIdempotencyConflict = errors.New("idempotency key conflicts with an existing request")
	ErrInvalidEntity       = errors.New("invalid domain entity")
)

type Workspace struct {
	ID        ID              `json:"id"`
	Name      string          `json:"name"`
	Status    WorkspaceStatus `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type Device struct {
	ID                  ID             `json:"id"`
	WorkspaceID         ID             `json:"workspace_id"`
	Vendor              string         `json:"vendor"`
	SerialNumber        string         `json:"serial_number"`
	GatewaySerialNumber string         `json:"gateway_serial_number,omitempty"`
	ProductModel        string         `json:"product_model,omitempty"`
	DeviceType          DeviceType     `json:"device_type"`
	Status              DeviceStatus   `json:"status"`
	Capabilities        map[string]any `json:"capabilities,omitempty"`
	RegisteredAt        time.Time      `json:"registered_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

type DeviceState struct {
	DeviceID       ID              `json:"device_id"`
	WorkspaceID    ID              `json:"workspace_id"`
	StateVersion   int64           `json:"state_version"`
	DeviceTime     *time.Time      `json:"device_time,omitempty"`
	ServerTime     time.Time       `json:"server_time"`
	Online         bool            `json:"online"`
	Latitude       *float64        `json:"latitude,omitempty"`
	Longitude      *float64        `json:"longitude,omitempty"`
	Altitude       *float64        `json:"altitude,omitempty"`
	BatteryPercent *float64        `json:"battery_percent,omitempty"`
	Mode           string          `json:"mode,omitempty"`
	Payload        json.RawMessage `json:"payload,omitempty"`
}

func (s *DeviceState) Apply(incoming DeviceState) error {
	if s.DeviceID != "" && (s.DeviceID != incoming.DeviceID || s.WorkspaceID != incoming.WorkspaceID) {
		return fmt.Errorf("%w: device or workspace mismatch", ErrInvalidEntity)
	}
	if incoming.StateVersion <= s.StateVersion {
		return ErrStaleState
	}
	*s = incoming
	return nil
}

type DeviceEvent struct {
	ID              ID              `json:"id"`
	EventID         string          `json:"event_id"`
	WorkspaceID     ID              `json:"workspace_id"`
	DeviceID        ID              `json:"device_id,omitempty"`
	GatewayDeviceID ID              `json:"gateway_device_id,omitempty"`
	EventType       string          `json:"event_type"`
	Method          string          `json:"method,omitempty"`
	DeviceTime      *time.Time      `json:"device_time,omitempty"`
	ReceivedAt      time.Time       `json:"received_at"`
	Sequence        *int64          `json:"sequence,omitempty"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

type Alarm struct {
	ID              ID              `json:"id"`
	WorkspaceID     ID              `json:"workspace_id"`
	DeviceID        ID              `json:"device_id"`
	DedupKey        string          `json:"dedup_key"`
	AlarmType       string          `json:"alarm_type"`
	Severity        AlarmSeverity   `json:"severity"`
	Status          AlarmStatus     `json:"status"`
	FirstOccurredAt time.Time       `json:"first_occurred_at"`
	LastOccurredAt  time.Time       `json:"last_occurred_at"`
	OccurrenceCount int64           `json:"occurrence_count"`
	AcknowledgedBy  string          `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	ResolvedAt      *time.Time      `json:"resolved_at,omitempty"`
	Details         json.RawMessage `json:"details,omitempty"`
}

func (a *Alarm) Acknowledge(actor string, at time.Time) error {
	if a.Status == AlarmResolved {
		return fmt.Errorf("%w: resolved alarm cannot be acknowledged", ErrInvalidTransition)
	}
	if a.Status == AlarmAcknowledged {
		return nil
	}
	if a.Status != AlarmOpen || actor == "" {
		return fmt.Errorf("%w: acknowledge requires OPEN status and actor", ErrInvalidEntity)
	}
	a.Status, a.AcknowledgedBy, a.AcknowledgedAt = AlarmAcknowledged, actor, &at
	return nil
}

func (a *Alarm) Resolve(at time.Time) error {
	if a.Status == AlarmResolved {
		return nil
	}
	if a.Status != AlarmOpen && a.Status != AlarmAcknowledged {
		return fmt.Errorf("%w: alarm is not active", ErrInvalidTransition)
	}
	a.Status, a.ResolvedAt = AlarmResolved, &at
	return nil
}

type Command struct {
	ID              ID              `json:"id"`
	WorkspaceID     ID              `json:"workspace_id"`
	TargetDeviceID  ID              `json:"target_device_id"`
	GatewayDeviceID ID              `json:"gateway_device_id,omitempty"`
	Method          string          `json:"method"`
	Status          CommandStatus   `json:"status"`
	RiskLevel       RiskLevel       `json:"risk_level"`
	IdempotencyKey  string          `json:"idempotency_key"`
	RequestHash     string          `json:"request_hash"`
	DJITID          string          `json:"dji_tid,omitempty"`
	DJIBID          string          `json:"dji_bid,omitempty"`
	Parameters      json.RawMessage `json:"parameters,omitempty"`
	RequestedBy     string          `json:"requested_by"`
	ResultCode      *int            `json:"result_code,omitempty"`
	ResultMessage   string          `json:"result_message,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	ExpiresAt       time.Time       `json:"expires_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CommandEvent struct {
	ID         ID            `json:"id"`
	CommandID  ID            `json:"command_id"`
	FromStatus CommandStatus `json:"from_status,omitempty"`
	ToStatus   CommandStatus `json:"to_status"`
	Source     string        `json:"source"`
	ResultCode *int          `json:"result_code,omitempty"`
	Message    string        `json:"message,omitempty"`
	OccurredAt time.Time     `json:"occurred_at"`
}

var commandTransitions = map[CommandStatus]map[CommandStatus]struct{}{
	CommandCreated:        {CommandValidated: {}, CommandRejected: {}, CommandCanceled: {}},
	CommandValidated:      {CommandPublishPending: {}, CommandCanceled: {}},
	CommandPublishPending: {CommandPublished: {}, CommandFailed: {}},
	CommandPublished:      {CommandAccepted: {}, CommandExecuting: {}, CommandSucceeded: {}, CommandFailed: {}, CommandTimeout: {}},
	CommandAccepted:       {CommandExecuting: {}, CommandSucceeded: {}, CommandFailed: {}, CommandTimeout: {}},
	CommandExecuting:      {CommandSucceeded: {}, CommandFailed: {}, CommandTimeout: {}},
}

func (c *Command) Transition(to CommandStatus, at time.Time, source, message string) (CommandEvent, error) {
	if c.Status == to {
		return CommandEvent{}, nil
	}
	if _, ok := commandTransitions[c.Status][to]; !ok {
		return CommandEvent{}, fmt.Errorf("%w: %s -> %s", ErrInvalidTransition, c.Status, to)
	}
	event := CommandEvent{CommandID: c.ID, FromStatus: c.Status, ToStatus: to, Source: source, Message: message, OccurredAt: at}
	c.Status, c.UpdatedAt = to, at
	switch to {
	case CommandSucceeded, CommandFailed, CommandTimeout, CommandCanceled, CommandRejected:
		c.CompletedAt = &at
	}
	return event, nil
}

func EnsureIdempotency(existingHash, requestedHash string) error {
	if existingHash != requestedHash {
		return ErrIdempotencyConflict
	}
	return nil
}

type OutboxEvent struct {
	ID            ID              `json:"id"`
	WorkspaceID   ID              `json:"workspace_id"`
	AggregateType string          `json:"aggregate_type"`
	AggregateID   ID              `json:"aggregate_id"`
	EventType     string          `json:"event_type"`
	Destination   string          `json:"destination"`
	Payload       json.RawMessage `json:"payload"`
	Status        string          `json:"status"`
	AttemptCount  int             `json:"attempt_count"`
	AvailableAt   time.Time       `json:"available_at"`
	LockedAt      *time.Time      `json:"locked_at,omitempty"`
	LockedBy      string          `json:"locked_by,omitempty"`
	PublishedAt   *time.Time      `json:"published_at,omitempty"`
	LastError     string          `json:"last_error,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}
