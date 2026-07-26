package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/iuoow/OpenDroneOps/internal/platform/logging"
	"github.com/iuoow/OpenDroneOps/internal/platform/signals"
	"github.com/iuoow/OpenDroneOps/internal/simulator"
)

func main() {
	logger := logging.New(os.Getenv("APP_ENV"))
	ctx, stop := signals.Context(context.Background())
	defer stop()
	config := simulator.DefaultConfig()
	sim, err := simulator.New(config)
	if err != nil {
		logger.Error("simulator configuration error", "error", err)
		os.Exit(2)
	}
	logger.Info("simulator started", "seed", config.Seed, "gateways", config.Gateways)
	if err := sim.Run(ctx, logPublisher{logger: logger}); err != nil && ctx.Err() == nil {
		logger.Error("simulator stopped with error", "error", err)
		os.Exit(1)
	}
	logger.Info("simulator stopped")
}

type logPublisher struct {
	logger *slog.Logger
}

func (p logPublisher) Publish(ctx context.Context, publication simulator.Publication) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.logger.Debug("simulator publication", "topic", publication.Topic, "qos", publication.QoS, "payload_hash", simulator.PayloadHash(publication.Payload))
	return nil
}
