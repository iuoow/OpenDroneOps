package simulator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var (
	ErrQueueFull      = errors.New("simulator publish queue is full")
	ErrUnknownMethod  = errors.New("simulator method is not allowed")
	ErrCommandTimeout = errors.New("simulator command timed out")
)

type Publisher interface {
	Publish(context.Context, Publication) error
}

type Publication struct {
	Topic   string
	Payload []byte
	QoS     byte
	Retain  bool
	At      time.Time
}

type FaultConfig struct {
	DuplicateRate      float64
	OutOfOrderRate     float64
	InvalidJSONRate    float64
	UnknownMethodRate  float64
	DisconnectRate     float64
	ReconnectDelay     time.Duration
	CommandFailureRate float64
	CommandTimeoutRate float64
}

type Config struct {
	Seed                int64
	Gateways            int
	AircraftPerGateway  int
	TelemetryInterval   time.Duration
	StateChangeInterval time.Duration
	QueueSize           int
	WorkspaceID         string
	QoS                 byte
	Faults              FaultConfig
	AllowedMethods      []string
}

func DefaultConfig() Config {
	return Config{
		Seed:                20260726,
		Gateways:            10,
		AircraftPerGateway:  1,
		TelemetryInterval:   time.Second,
		StateChangeInterval: 30 * time.Second,
		QueueSize:           256,
		WorkspaceID:         "00000000-0000-0000-0000-000000000001",
		QoS:                 1,
		AllowedMethods:      []string{"sim_status_refresh", "sim_alarm_trigger", "sim_alarm_resolve"},
	}
}

type CommandRequest struct {
	GatewaySN string
	TID       string
	BID       string
	Method    string
	Data      json.RawMessage
}

type CommandOutcome string

const (
	CommandSucceeded CommandOutcome = "SUCCEEDED"
	CommandFailed    CommandOutcome = "FAILED"
	CommandTimedOut  CommandOutcome = "TIMEOUT"
)

type CommandResult struct {
	Outcome      CommandOutcome
	Latency      time.Duration
	Publications []Publication
}

type Simulator struct {
	config          Config
	rng             *rand.Rand
	mu              sync.Mutex
	seq             map[string]int64
	offlineUntil    map[string]time.Time
	lastStateChange time.Time
}

func New(config Config) (*Simulator, error) {
	if config.Gateways < 1 || config.AircraftPerGateway < 1 || config.QueueSize < 1 {
		return nil, errors.New("gateways, aircraft per gateway, and queue size must be positive")
	}
	if config.TelemetryInterval <= 0 || config.StateChangeInterval <= 0 {
		return nil, errors.New("simulator intervals must be positive")
	}
	if config.QoS > 2 {
		return nil, errors.New("MQTT QoS must be between 0 and 2")
	}
	return &Simulator{
		config:       config,
		rng:          rand.New(rand.NewSource(config.Seed)),
		seq:          make(map[string]int64),
		offlineUntil: make(map[string]time.Time),
	}, nil
}

func (s *Simulator) Run(ctx context.Context, publisher Publisher) error {
	ticker := time.NewTicker(s.config.TelemetryInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-ticker.C:
			if err := s.PublishTick(ctx, publisher, now); err != nil {
				return err
			}
		}
	}
}

func (s *Simulator) PublishTick(ctx context.Context, publisher Publisher, now time.Time) error {
	publications := s.Tick(now)
	if len(publications) > s.config.QueueSize {
		return ErrQueueFull
	}
	for _, publication := range publications {
		if err := publisher.Publish(ctx, publication); err != nil {
			return err
		}
	}
	return nil
}

func (s *Simulator) Tick(now time.Time) []Publication {
	s.mu.Lock()
	defer s.mu.Unlock()
	publications := make([]Publication, 0, s.config.Gateways*(s.config.AircraftPerGateway+1))
	for gateway := 1; gateway <= s.config.Gateways; gateway++ {
		gatewaySN := fmt.Sprintf("SIM-GW-%03d", gateway)
		online := s.gatewayOnline(gatewaySN, now)
		for aircraft := 1; aircraft <= s.config.AircraftPerGateway; aircraft++ {
			deviceSN := fmt.Sprintf("%s-AIR-%03d", gatewaySN, aircraft)
			if !online {
				continue
			}
			sequence := s.nextSequence(deviceSN)
			publications = append(publications,
				s.telemetry("thing/product/"+deviceSN+"/osd", gatewaySN, "sim/osd", deviceSN, sequence, true, now),
				s.telemetry("thing/product/"+deviceSN+"/state", gatewaySN, "sim/state", deviceSN, sequence, true, now),
			)
			if s.lastStateChange.IsZero() || now.Sub(s.lastStateChange) >= s.config.StateChangeInterval {
				publications = append(publications, s.telemetry("thing/product/"+gatewaySN+"/events", gatewaySN, "sim/state_change", deviceSN, sequence, true, now))
			}
		}
		publications = append(publications, s.telemetry("sys/product/"+gatewaySN+"/status", gatewaySN, "sim/status", gatewaySN, s.nextSequence(gatewaySN), online, now))
	}
	if s.lastStateChange.IsZero() || now.Sub(s.lastStateChange) >= s.config.StateChangeInterval {
		s.lastStateChange = now
	}
	return s.injectFaults(publications)
}

func (s *Simulator) HandleCommand(ctx context.Context, request CommandRequest, now time.Time) (CommandResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.allowed(request.Method) {
		if s.rng.Float64() >= s.config.Faults.UnknownMethodRate {
			return CommandResult{}, fmt.Errorf("%w: %s", ErrUnknownMethod, request.Method)
		}
	}
	if err := ctx.Err(); err != nil {
		return CommandResult{}, err
	}
	latency := 100 * time.Millisecond
	if s.config.Faults.CommandTimeoutRate > 0 && s.rng.Float64() < s.config.Faults.CommandTimeoutRate {
		return CommandResult{Outcome: CommandTimedOut, Latency: latency, Publications: nil}, ErrCommandTimeout
	}
	outcome := CommandSucceeded
	resultCode := 0
	if s.config.Faults.CommandFailureRate > 0 && s.rng.Float64() < s.config.Faults.CommandFailureRate {
		outcome, resultCode = CommandFailed, 500
	}
	payload := s.commandReply(request, now, string(outcome), resultCode)
	publication := Publication{
		Topic:   "thing/product/" + request.GatewaySN + "/services_reply",
		Payload: payload, QoS: s.config.QoS, At: now.Add(latency),
	}
	return CommandResult{Outcome: outcome, Latency: latency, Publications: []Publication{publication}}, nil
}

func (s *Simulator) telemetry(topic, gateway, method, device string, sequence int64, online bool, now time.Time) Publication {
	data := map[string]any{"device_sn": device, "workspace_id": s.config.WorkspaceID, "online": online}
	payload, _ := json.Marshal(map[string]any{
		"tid": fmt.Sprintf("tid-%s-%d", device, sequence), "bid": fmt.Sprintf("bid-%d", sequence),
		"timestamp": now.UnixMilli(), "gateway": gateway, "method": method, "seq": sequence, "data": data,
	})
	return Publication{Topic: topic, Payload: payload, QoS: s.config.QoS, At: now}
}

func (s *Simulator) gatewayOnline(gateway string, now time.Time) bool {
	if until, ok := s.offlineUntil[gateway]; ok {
		if now.Before(until) {
			return false
		}
		delete(s.offlineUntil, gateway)
		return true
	}
	if s.config.Faults.DisconnectRate > 0 && s.rng.Float64() < s.config.Faults.DisconnectRate {
		delay := s.config.Faults.ReconnectDelay
		if delay <= 0 {
			delay = time.Second
		}
		s.offlineUntil[gateway] = now.Add(delay)
		return false
	}
	return true
}

func (s *Simulator) commandReply(request CommandRequest, now time.Time, outcome string, resultCode int) []byte {
	payload, _ := json.Marshal(map[string]any{
		"tid": request.TID, "bid": request.BID, "timestamp": now.UnixMilli(),
		"gateway": request.GatewaySN, "method": request.Method,
		"data": map[string]any{"result": outcome, "result_code": resultCode},
	})
	return payload
}

func (s *Simulator) injectFaults(publications []Publication) []Publication {
	result := make([]Publication, 0, len(publications)*2)
	for _, publication := range publications {
		if s.rng.Float64() < s.config.Faults.InvalidJSONRate {
			publication.Payload = []byte(`{"invalid_json":`)
		}
		if s.rng.Float64() < s.config.Faults.OutOfOrderRate {
			publication.Payload = lowerSequence(publication.Payload)
		}
		result = append(result, publication)
		if s.rng.Float64() < s.config.Faults.DuplicateRate {
			result = append(result, publication)
		}
	}
	return result
}

func lowerSequence(payload []byte) []byte {
	var envelope map[string]any
	if json.Unmarshal(payload, &envelope) != nil {
		return payload
	}
	if sequence, ok := envelope["seq"].(float64); ok && sequence > 0 {
		envelope["seq"] = int64(sequence) - 1
	}
	updated, _ := json.Marshal(envelope)
	return updated
}

func (s *Simulator) nextSequence(device string) int64 {
	s.seq[device]++
	return s.seq[device]
}

func (s *Simulator) allowed(method string) bool {
	for _, allowed := range s.config.AllowedMethods {
		if method == allowed {
			return true
		}
	}
	return false
}

func PayloadHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
