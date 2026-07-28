package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/buildinfo"
	"github.com/iuoow/OpenDroneOps/internal/config"
	"github.com/iuoow/OpenDroneOps/internal/httpapi"
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
	logger.Info("server starting", "build", buildinfo.Current().String())
	ctx, stop := signals.Context(context.Background())
	defer stop()

	server := httpapi.NewWithOptions(cfg.HTTPAddr, logger, []httpapi.Option{
		httpapi.WithAdminAddress(cfg.AdminAddr),
	})
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- server.Run()
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			logger.Error("http server stopped", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown failed", "error", err)
			os.Exit(1)
		}
		logger.Info("server stopped")
	}
}
