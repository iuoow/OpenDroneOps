package command

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/iuoow/OpenDroneOps/internal/protocol/dji"
	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

var (
	ErrUnknownMethod  = errors.New("command method is not registered")
	ErrInvalidRequest = errors.New("invalid command request")
)

type MethodDefinition struct {
	Name      string
	Timeout   time.Duration
	RiskLevel domain.RiskLevel
	QoS       byte
}

type Registry struct {
	methods map[string]MethodDefinition
}

func NewRegistry(definitions []MethodDefinition) (*Registry, error) {
	registry := &Registry{methods: make(map[string]MethodDefinition, len(definitions))}
	for _, definition := range definitions {
		if definition.Name == "" || definition.Timeout <= 0 || definition.QoS > 2 {
			return nil, fmt.Errorf("%w: invalid method definition", ErrInvalidRequest)
		}
		if definition.RiskLevel != domain.RiskLow && definition.RiskLevel != domain.RiskMedium && definition.RiskLevel != domain.RiskHigh {
			return nil, fmt.Errorf("%w: invalid risk level", ErrInvalidRequest)
		}
		if _, exists := registry.methods[definition.Name]; exists {
			return nil, fmt.Errorf("%w: duplicate method %s", ErrInvalidRequest, definition.Name)
		}
		registry.methods[definition.Name] = definition
	}
	return registry, nil
}

func DefaultRegistry() *Registry {
	registry, _ := NewRegistry([]MethodDefinition{{
		Name: "sim_status_refresh", Timeout: 30 * time.Second, RiskLevel: domain.RiskLow, QoS: 1,
	}})
	return registry
}

func (r *Registry) Lookup(method string) (MethodDefinition, bool) {
	if r == nil {
		return MethodDefinition{}, false
	}
	definition, ok := r.methods[method]
	return definition, ok
}

type CreateRequest struct {
	WorkspaceID     domain.ID
	TargetDeviceID  domain.ID
	GatewayDeviceID domain.ID
	GatewaySN       string
	Method          string
	Parameters      json.RawMessage
	IdempotencyKey  string
	RequestedBy     string
	RequestID       string
	ExpiresAt       time.Time
}

type CreateBundle struct {
	Command domain.Command
	Events  []domain.CommandEvent
	Outbox  domain.OutboxEvent
	Audit   AuditRecord
}

type Reply struct {
	WorkspaceID  domain.ID
	TID          string
	BID          string
	Method       string
	GatewaySN    string
	Status       domain.CommandStatus
	ResultCode   *int
	Message      string
	RawMessageID domain.ID
	PayloadHash  string
	Payload      json.RawMessage
	ReceivedAt   time.Time
}

type AuditRecord struct {
	WorkspaceID  domain.ID
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   domain.ID
	RequestID    string
	Details      json.RawMessage
	CreatedAt    time.Time
}

type OrphanReply struct {
	WorkspaceID domain.ID
	TID         string
	BID         string
	Method      string
	GatewaySN   string
	PayloadHash string
	Payload     json.RawMessage
	ReceivedAt  time.Time
}

type Delivery struct {
	Event     domain.OutboxEvent
	RiskLevel domain.RiskLevel
	QoS       byte
}

type Repository interface {
	Create(context.Context, CreateBundle) (domain.Command, bool, error)
	LeaseOutbox(context.Context, string, int, time.Time, time.Duration) ([]Delivery, error)
	MarkPublished(context.Context, string, domain.ID, domain.ID, time.Time) (domain.Command, bool, error)
	MarkRetry(context.Context, string, domain.ID, time.Time, string) error
	MarkFailed(context.Context, string, domain.ID, domain.ID, time.Time, string) (domain.Command, bool, error)
	ApplyReply(context.Context, Reply) (domain.Command, bool, bool, error)
	RecordOrphanReply(context.Context, OrphanReply) (bool, error)
	Expire(context.Context, time.Time, int) ([]domain.Command, error)
}

type Publisher interface {
	Publish(websockethub.Event)
}

type Service struct {
	repository Repository
	registry   *Registry
	publisher  Publisher
	now        func() time.Time
}

func NewService(repository Repository, registry *Registry, publisher Publisher) (*Service, error) {
	if repository == nil {
		return nil, errors.New("command repository is required")
	}
	if registry == nil {
		registry = DefaultRegistry()
	}
	return &Service{repository: repository, registry: registry, publisher: publisher, now: func() time.Time {
		return time.Now().UTC()
	}}, nil
}

func (s *Service) Create(ctx context.Context, request CreateRequest) (domain.Command, bool, error) {
	definition, ok := s.registry.Lookup(request.Method)
	if !ok {
		return domain.Command{}, false, fmt.Errorf("%w: %s", ErrUnknownMethod, request.Method)
	}
	if err := validateCreateRequest(request); err != nil {
		return domain.Command{}, false, err
	}
	parameters, err := canonicalJSON(request.Parameters)
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("%w: parameters: %v", ErrInvalidRequest, err)
	}
	now := s.now()
	expiresAt := request.ExpiresAt
	if expiresAt.IsZero() {
		expiresAt = now.Add(definition.Timeout)
	}
	if !expiresAt.After(now) {
		return domain.Command{}, false, fmt.Errorf("%w: expires_at must be in the future", ErrInvalidRequest)
	}
	requestHash, err := hashRequest(request, parameters)
	if err != nil {
		return domain.Command{}, false, err
	}
	commandID, err := newUUID()
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("generate command id: %w", err)
	}
	tid, err := newUUID()
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("generate DJI tid: %w", err)
	}
	bid, err := newUUID()
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("generate DJI bid: %w", err)
	}
	command := domain.Command{
		ID: commandID, WorkspaceID: request.WorkspaceID, TargetDeviceID: request.TargetDeviceID,
		GatewayDeviceID: request.GatewayDeviceID, Method: request.Method, Status: domain.CommandCreated,
		RiskLevel: definition.RiskLevel, IdempotencyKey: request.IdempotencyKey, RequestHash: requestHash,
		DJITID: string(tid), DJIBID: string(bid), Parameters: parameters, RequestedBy: request.RequestedBy,
		CreatedAt: now, UpdatedAt: now, ExpiresAt: expiresAt,
	}
	createdEventID, err := newUUID()
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("generate command event id: %w", err)
	}
	events := []domain.CommandEvent{{
		ID: createdEventID, CommandID: command.ID, ToStatus: domain.CommandCreated,
		Source: "api", OccurredAt: now,
	}}
	for _, status := range []domain.CommandStatus{domain.CommandValidated, domain.CommandPublishPending} {
		event, transitionErr := command.Transition(status, now, "api", "")
		if transitionErr != nil {
			return domain.Command{}, false, transitionErr
		}
		event.ID, err = newUUID()
		if err != nil {
			return domain.Command{}, false, fmt.Errorf("generate command event id: %w", err)
		}
		events = append(events, event)
	}
	envelope, err := json.Marshal(map[string]any{
		"tid": tid, "bid": bid, "timestamp": now.UnixMilli(), "gateway": request.GatewaySN,
		"method": request.Method, "data": json.RawMessage(parameters),
	})
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("encode DJI service envelope: %w", err)
	}
	outboxID, err := newUUID()
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("generate outbox id: %w", err)
	}
	outbox := domain.OutboxEvent{
		ID: outboxID, WorkspaceID: request.WorkspaceID, AggregateType: "COMMAND",
		AggregateID: command.ID, EventType: "command.publish", Destination: "thing/product/" + request.GatewaySN + "/services",
		Payload: envelope, Status: "PENDING", AvailableAt: now, CreatedAt: now,
	}
	auditDetails, _ := json.Marshal(map[string]any{"method": request.Method, "risk_level": definition.RiskLevel})
	bundle := CreateBundle{Command: command, Events: events, Outbox: outbox, Audit: AuditRecord{
		WorkspaceID: request.WorkspaceID, ActorID: request.RequestedBy, Action: "command.create",
		ResourceType: "command", ResourceID: command.ID, RequestID: request.RequestID,
		Details: auditDetails, CreatedAt: now,
	}}
	persisted, created, err := s.repository.Create(ctx, bundle)
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("create command transaction: %w", err)
	}
	if created {
		s.publish(persisted, now)
	}
	return persisted, created, nil
}

func (s *Service) HandleDJIReply(ctx context.Context, workspaceID domain.ID, message dji.Message, rawMessageID domain.ID, receivedAt time.Time) (domain.Command, bool, error) {
	if message.Topic.Kind != dji.TopicServicesReply {
		return domain.Command{}, false, fmt.Errorf("%w: topic is not services_reply", ErrInvalidRequest)
	}
	if message.Envelope.TID == "" || message.Envelope.BID == "" || message.Envelope.Method == "" {
		return domain.Command{}, false, fmt.Errorf("%w: reply requires tid, bid, and method", ErrInvalidRequest)
	}
	if receivedAt.IsZero() {
		receivedAt = s.now()
	}
	status, resultCode, resultMessage, err := decodeReplyData(message.Envelope.Data)
	if err != nil {
		return domain.Command{}, false, err
	}
	reply := Reply{
		WorkspaceID: workspaceID, TID: message.Envelope.TID, BID: message.Envelope.BID,
		Method: message.Envelope.Method, GatewaySN: message.Topic.GatewaySN, Status: status,
		ResultCode: resultCode, Message: resultMessage, RawMessageID: rawMessageID,
		PayloadHash: hex.EncodeToString(message.PayloadHash[:]), Payload: message.Envelope.Data, ReceivedAt: receivedAt,
	}
	command, found, changed, err := s.repository.ApplyReply(ctx, reply)
	if err != nil {
		return domain.Command{}, false, fmt.Errorf("apply DJI command reply: %w", err)
	}
	if !found {
		_, orphanErr := s.repository.RecordOrphanReply(ctx, OrphanReply{
			WorkspaceID: workspaceID, TID: reply.TID, BID: reply.BID, Method: reply.Method,
			GatewaySN: reply.GatewaySN, PayloadHash: reply.PayloadHash, Payload: reply.Payload, ReceivedAt: receivedAt,
		})
		if orphanErr != nil {
			return domain.Command{}, false, fmt.Errorf("persist orphan command reply: %w", orphanErr)
		}
		return domain.Command{}, false, nil
	}
	if changed {
		s.publish(command, receivedAt)
	}
	return command, changed, nil
}

func (s *Service) Expire(ctx context.Context, limit int) ([]domain.Command, error) {
	if limit < 1 {
		limit = 100
	}
	now := s.now()
	commands, err := s.repository.Expire(ctx, now, limit)
	if err != nil {
		return nil, fmt.Errorf("expire commands: %w", err)
	}
	for _, command := range commands {
		s.publish(command, now)
	}
	return commands, nil
}

func (s *Service) NotifyTransition(command domain.Command, occurredAt time.Time) {
	s.publish(command, occurredAt)
}

func (s *Service) publish(command domain.Command, occurredAt time.Time) {
	if s.publisher == nil {
		return
	}
	data, err := json.Marshal(command)
	if err != nil {
		return
	}
	s.publisher.Publish(websockethub.Event{
		EventID: fmt.Sprintf("command:%s:%s:%d", command.ID, command.Status, occurredAt.UnixNano()),
		Type:    "command.updated", SchemaVersion: "1.0", WorkspaceID: string(command.WorkspaceID),
		AggregateID: string(command.TargetDeviceID), OccurredAt: occurredAt, Data: data,
	})
}

func validateCreateRequest(request CreateRequest) error {
	if request.WorkspaceID == "" || request.TargetDeviceID == "" || request.GatewayDeviceID == "" ||
		strings.TrimSpace(request.GatewaySN) == "" || strings.TrimSpace(request.Method) == "" ||
		len(request.IdempotencyKey) < 8 || strings.TrimSpace(request.RequestedBy) == "" {
		return ErrInvalidRequest
	}
	if strings.ContainsAny(request.GatewaySN, "/+#") {
		return fmt.Errorf("%w: invalid gateway serial number", ErrInvalidRequest)
	}
	return nil
}

func canonicalJSON(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) == 0 {
		return json.RawMessage(`{}`), nil
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return json.Marshal(value)
}

func hashRequest(request CreateRequest, parameters json.RawMessage) (string, error) {
	payload, err := json.Marshal(struct {
		WorkspaceID     domain.ID       `json:"workspace_id"`
		TargetDeviceID  domain.ID       `json:"target_device_id"`
		GatewayDeviceID domain.ID       `json:"gateway_device_id"`
		GatewaySN       string          `json:"gateway_sn"`
		Method          string          `json:"method"`
		Parameters      json.RawMessage `json:"parameters"`
		ExpiresAt       time.Time       `json:"expires_at,omitempty"`
	}{
		WorkspaceID: request.WorkspaceID, TargetDeviceID: request.TargetDeviceID,
		GatewayDeviceID: request.GatewayDeviceID, GatewaySN: request.GatewaySN,
		Method: request.Method, Parameters: parameters, ExpiresAt: request.ExpiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("hash command request: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

func decodeReplyData(data json.RawMessage) (domain.CommandStatus, *int, string, error) {
	var reply struct {
		Result     string `json:"result"`
		ResultCode *int   `json:"result_code"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(data, &reply); err != nil {
		return "", nil, "", fmt.Errorf("%w: invalid reply data: %v", ErrInvalidRequest, err)
	}
	var status domain.CommandStatus
	switch strings.ToUpper(reply.Result) {
	case "ACCEPTED":
		status = domain.CommandAccepted
	case "EXECUTING":
		status = domain.CommandExecuting
	case "SUCCEEDED", "SUCCESS":
		status = domain.CommandSucceeded
	case "FAILED", "FAILURE":
		status = domain.CommandFailed
	default:
		if reply.ResultCode != nil && *reply.ResultCode == 0 {
			status = domain.CommandSucceeded
		} else if reply.ResultCode != nil {
			status = domain.CommandFailed
		} else {
			return "", nil, "", fmt.Errorf("%w: unsupported reply result %q", ErrInvalidRequest, reply.Result)
		}
	}
	return status, reply.ResultCode, reply.Message, nil
}

func newUUID() (domain.ID, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return domain.ID(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16])), nil
}
