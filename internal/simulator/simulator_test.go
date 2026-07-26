package simulator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/protocol/dji"
)

func TestTickIsDeterministicAndUsesExpectedTopics(t *testing.T) {
	config := DefaultConfig()
	config.Gateways, config.AircraftPerGateway = 1, 1
	first, _ := New(config)
	second, _ := New(config)
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	left, right := first.Tick(now), second.Tick(now)
	if len(left) != 4 || len(right) != 4 {
		t.Fatalf("expected OSD, State, Event and Status, got %d and %d", len(left), len(right))
	}
	for i := range left {
		if left[i].Topic != right[i].Topic || string(left[i].Payload) != string(right[i].Payload) {
			t.Fatalf("seeded simulators diverged at %d", i)
		}
		if _, err := dji.ParseTopic(left[i].Topic); err != nil {
			t.Fatalf("generated unsupported topic %q: %v", left[i].Topic, err)
		}
	}
}

func TestFaultInjectionDuplicateInvalidAndOutOfOrder(t *testing.T) {
	config := DefaultConfig()
	config.Gateways, config.AircraftPerGateway = 1, 1
	config.Faults.DuplicateRate, config.Faults.InvalidJSONRate, config.Faults.OutOfOrderRate = 1, 1, 1
	sim, _ := New(config)
	publications := sim.Tick(time.Unix(0, 0))
	if len(publications) != 8 {
		t.Fatalf("expected every publication to be duplicated, got %d", len(publications))
	}
	if string(publications[0].Payload) != `{"invalid_json":` {
		t.Fatalf("invalid JSON fault not injected: %s", publications[0].Payload)
	}
	config.Faults.DuplicateRate, config.Faults.InvalidJSONRate = 0, 0
	orderedSim, _ := New(config)
	ordered := orderedSim.Tick(time.Unix(0, 0))
	envelope, err := dji.DecodeEnvelope(ordered[0].Payload)
	if err != nil || envelope.Seq == nil || *envelope.Seq != 0 {
		t.Fatalf("out-of-order sequence fault not injected: envelope=%+v err=%v", envelope, err)
	}
}

func TestHandleCommandOutcomesAndAllowlist(t *testing.T) {
	config := DefaultConfig()
	config.Faults.CommandFailureRate = 1
	sim, _ := New(config)
	result, err := sim.HandleCommand(context.Background(), CommandRequest{
		GatewaySN: "SIM-GW-001", TID: "tid-1", BID: "bid-1", Method: "sim_status_refresh",
	}, time.Unix(0, 0))
	if err != nil || result.Outcome != CommandFailed || len(result.Publications) != 1 {
		t.Fatalf("failed command simulation incorrect: result=%+v err=%v", result, err)
	}
	if _, err := dji.DecodeEnvelope(result.Publications[0].Payload); err != nil {
		t.Fatalf("command reply is not a valid envelope: %v", err)
	}
	if _, err := sim.HandleCommand(context.Background(), CommandRequest{Method: "not-allowed"}, time.Unix(0, 0)); !errors.Is(err, ErrUnknownMethod) {
		t.Fatalf("unknown method error = %v", err)
	}
}

func TestDisconnectAndReconnectLifecycle(t *testing.T) {
	config := DefaultConfig()
	config.Gateways, config.AircraftPerGateway = 1, 1
	config.Faults.DisconnectRate = 1
	config.Faults.ReconnectDelay = time.Second
	sim, _ := New(config)
	first := sim.Tick(time.Unix(0, 0))
	if len(first) != 1 || string(first[0].Payload) == "" {
		t.Fatalf("offline tick should retain gateway status publication: %+v", first)
	}
	second := sim.Tick(time.Unix(2, 0))
	if len(second) != 3 {
		t.Fatalf("reconnected tick should include telemetry and status, got %d", len(second))
	}
}

func TestCommandTimeoutIsFiniteAndCancellable(t *testing.T) {
	config := DefaultConfig()
	config.Faults.CommandTimeoutRate = 1
	sim, _ := New(config)
	result, err := sim.HandleCommand(context.Background(), CommandRequest{
		GatewaySN: "SIM-GW-001", Method: "sim_status_refresh",
	}, time.Unix(0, 0))
	if !errors.Is(err, ErrCommandTimeout) || result.Outcome != CommandTimedOut || len(result.Publications) != 0 {
		t.Fatalf("timeout outcome incorrect: result=%+v err=%v", result, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = sim.HandleCommand(ctx, CommandRequest{GatewaySN: "SIM-GW-001", Method: "sim_status_refresh"}, time.Unix(0, 0))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled command returned %v", err)
	}
}

func TestRunStopsOnContextAndQueueIsBounded(t *testing.T) {
	config := DefaultConfig()
	config.Gateways, config.AircraftPerGateway, config.QueueSize = 1, 1, 1
	sim, _ := New(config)
	if err := sim.PublishTick(context.Background(), nopPublisher{}, time.Unix(0, 0)); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected bounded queue error, got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sim.Run(ctx, nopPublisher{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled run, got %v", err)
	}
}

type nopPublisher struct{}

func (nopPublisher) Publish(context.Context, Publication) error { return nil }
