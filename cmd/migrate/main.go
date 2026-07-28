package main

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/buildinfo"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func main() {
	slog.Info("migration starting", "build", buildinfo.Current().String())
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		slog.Error("configuration error", "error", "POSTGRES_DSN is required")
		os.Exit(2)
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		slog.Error("open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("ping database", "error", err)
		os.Exit(1)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("set migration dialect", "error", err)
		os.Exit(1)
	}
	if err := goose.Up(db, "db/migrations"); err != nil {
		slog.Error("apply migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations applied")
}
