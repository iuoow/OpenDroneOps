package config

import (
	"os"
	"strings"
	"testing"
)

func TestLoadRequiresPostgresDSN(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "")
	t.Setenv("MQTT_URL", "mqtt://localhost:1883")
	t.Setenv("MQTT_CLIENT_ID", "test-client")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_DSN") {
		t.Fatalf("expected missing POSTGRES_DSN error, got %v", err)
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("POSTGRES_DSN", "postgres://local")
	t.Setenv("MQTT_URL", "mqtt://localhost:1883")
	t.Setenv("MQTT_CLIENT_ID", "test-client")
	_ = os.Unsetenv("WS_SEND_QUEUE_SIZE")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.WebSocketSendQueueSize != 256 {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}
