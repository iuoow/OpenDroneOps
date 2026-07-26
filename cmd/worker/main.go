package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/config"
	"github.com/iuoow/OpenDroneOps/internal/mqttworker"
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
	quarantine := &mqttworker.MemoryQuarantine{}
	ingestion, err := mqttworker.New(mqttworker.Config{
		ShardCount: cfg.MQTTShardCount, QueueSize: cfg.MQTTShardQueueSize,
		OnError: func(err error) { logger.Error("mqtt worker error", "error", err) },
	}, logHandler{logger: logger}, mqttworker.NewMemoryDeduplicator(), quarantine)
	if err != nil {
		logger.Error("worker configuration error", "error", err)
		os.Exit(2)
	}
	if err := ingestion.Start(ctx); err != nil {
		logger.Error("worker start error", "error", err)
		os.Exit(1)
	}
	broker, err := mqttworker.ConnectBroker(ctx, mqttworker.BrokerConfig{
		URL: cfg.MQTTURL, ClientID: cfg.MQTTClientID,
		Username: cfg.MQTTUsername, Password: cfg.MQTTPassword,
		KeepAlive: cfg.MQTTKeepAlive, SessionExpiry: cfg.MQTTSessionExpiry,
		OnConnectError: func(err error) { logger.Warn("mqtt connection error", "error", err) },
	}, func(message mqttworker.RawMessage) {
		if err := ingestion.Enqueue(ctx, message); err != nil {
			logger.Warn("mqtt message rejected", "topic", message.Topic, "error", err)
		}
	})
	if err != nil {
		logger.Error("mqtt broker setup error", "error", err)
		_ = ingestion.Close()
		os.Exit(1)
	}
	logger.Info("worker started", "mqtt_url", cfg.MQTTURL, "shards", cfg.MQTTShardCount)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := broker.Close(shutdownCtx); err != nil {
		logger.Warn("mqtt broker shutdown error", "error", err)
	}
	_ = ingestion.Close()
	logger.Info("worker stopped", "stats", ingestion.Stats())
}

type logHandler struct {
	logger *slog.Logger
}

func (h logHandler) Handle(ctx context.Context, message mqttworker.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	h.logger.Debug("mqtt message handled", "topic", message.Raw.Topic, "dedup_key", message.DedupKey, "attempt", message.Attempts)
	return nil
}
