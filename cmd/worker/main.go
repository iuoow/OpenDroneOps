package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/iuoow/OpenDroneOps/internal/config"
	"github.com/iuoow/OpenDroneOps/internal/platform/logging"
	"github.com/iuoow/OpenDroneOps/internal/platform/signals"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(2)
	}
	logger := logging.New(cfg.AppEnv)
	ctx, stop := signals.Context(context.Background())
	defer stop()
	logger.Info("worker scaffold started", "mqtt_url", cfg.MQTTURL, "shards", cfg.MQTTShardCount)
	<-ctx.Done()
	logger.Info("worker scaffold stopped")
}
