//go:build integration

package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/iuoow/OpenDroneOps/internal/mqttworker"
	"github.com/iuoow/OpenDroneOps/internal/twin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/redis/go-redis/v9"
)

const (
	integrationWorkspaceID = "11111111-1111-1111-1111-111111111111"
	integrationDeviceID    = "22222222-2222-2222-2222-222222222222"
)

func TestPostgresRedisAndMosquittoE2E(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("set INTEGRATION_TEST=1 after starting local PostgreSQL, Redis, and Mosquitto")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dsn := requiredEnv(t, "POSTGRES_DSN")
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping postgres: %v", err)
	}
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set goose dialect: %v", err)
	}
	if err := goose.UpContext(ctx, db, "../db/migrations"); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	prepareDevice(t, ctx, db)
	t.Cleanup(func() { cleanupDevice(ctx, db) })

	redisClient := redis.NewClient(&redis.Options{Addr: requiredEnv(t, "REDIS_ADDR"), Password: os.Getenv("REDIS_PASSWORD")})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	cache, err := twin.NewRedisCache(redisClient, "opendroneops-e2e")
	if err != nil {
		t.Fatal(err)
	}
	repository, err := twin.NewPostgresRepository(db)
	if err != nil {
		t.Fatal(err)
	}
	service, err := twin.NewService(repository, repository, cache, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	latitude, longitude, battery := 31.2304, 121.4737, 74.0
	state := domain.DeviceState{
		WorkspaceID: integrationWorkspaceID,
		DeviceID:    integrationDeviceID, StateVersion: 1, ServerTime: time.Now().UTC(), Online: true,
		Latitude: &latitude, Longitude: &longitude, BatteryPercent: &battery, Mode: "E2E",
	}
	result, err := service.ApplyState(ctx, state)
	if err != nil || !result.Accepted || !result.CacheUpdated {
		t.Fatalf("apply state result=%+v err=%v", result, err)
	}
	persisted, err := repository.GetLatest(ctx, integrationWorkspaceID, integrationDeviceID)
	if err != nil || persisted.StateVersion != 1 {
		t.Fatalf("postgres latest=%+v err=%v", persisted, err)
	}
	cached, err := cache.GetLatest(ctx, integrationWorkspaceID, integrationDeviceID)
	if err != nil || cached.StateVersion != 1 {
		t.Fatalf("redis latest=%+v err=%v", cached, err)
	}

	event := domain.DeviceEvent{
		EventID: "e2e-event-001", WorkspaceID: integrationWorkspaceID, DeviceID: integrationDeviceID,
		EventType: "device.telemetry", Method: "e2e/osd", ReceivedAt: time.Now().UTC(),
	}
	accepted, err := service.RecordEvent(ctx, event)
	if err != nil || !accepted {
		t.Fatalf("first event accepted=%v err=%v", accepted, err)
	}
	accepted, err = service.RecordEvent(ctx, event)
	if err != nil || accepted {
		t.Fatalf("duplicate event accepted=%v err=%v", accepted, err)
	}

	received := make(chan mqttworker.RawMessage, 1)
	clientID := fmt.Sprintf("opendroneops-e2e-%d", time.Now().UnixNano())
	broker, err := mqttworker.ConnectBroker(ctx, mqttworker.BrokerConfig{
		URL: requiredEnv(t, "MQTT_URL"), ClientID: clientID, KeepAlive: 5 * time.Second,
		SessionExpiry: time.Minute, CleanStart: true,
	}, func(message mqttworker.RawMessage) {
		if strings.Contains(message.Topic, integrationDeviceID) {
			select {
			case received <- message:
			default:
			}
		}
	})
	if err != nil {
		t.Fatalf("connect mosquitto: %v", err)
	}
	defer broker.Close(context.Background())
	topic := "thing/product/" + integrationDeviceID + "/osd"
	if err := broker.Publish(ctx, topic, []byte(`{"bid":"e2e","method":"e2e/osd"}`), 1); err != nil {
		t.Fatalf("publish mqtt: %v", err)
	}
	select {
	case message := <-received:
		if message.Topic != topic || string(message.Payload) == "" {
			t.Fatalf("received unexpected mqtt message: %+v", message)
		}
	case <-ctx.Done():
		t.Fatal("did not receive published mqtt message before timeout")
	}
}

func requiredEnv(t *testing.T, name string) string {
	t.Helper()
	value := os.Getenv(name)
	if value == "" {
		t.Fatalf("%s is required for integration test", name)
	}
	return value
}

func prepareDevice(t *testing.T, ctx context.Context, db *sql.DB) {
	t.Helper()
	cleanupDevice(ctx, db)
	if _, err := db.ExecContext(ctx, `INSERT INTO workspaces (id, name, status) VALUES ($1, 'E2E Workspace', 'ACTIVE')`, integrationWorkspaceID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO devices (id, workspace_id, vendor, serial_number, device_type, status)
VALUES ($1, $2, 'E2E', 'E2E-AIR-001', 'AIRCRAFT', 'ONLINE')`, integrationDeviceID, integrationWorkspaceID); err != nil {
		t.Fatalf("insert device: %v", err)
	}
}

func cleanupDevice(ctx context.Context, db *sql.DB) {
	_, _ = db.ExecContext(ctx, `DELETE FROM device_events WHERE workspace_id = $1`, integrationWorkspaceID)
	_, _ = db.ExecContext(ctx, `DELETE FROM device_latest_states WHERE device_id = $1`, integrationDeviceID)
	_, _ = db.ExecContext(ctx, `DELETE FROM devices WHERE id = $1`, integrationDeviceID)
	_, _ = db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = $1`, integrationWorkspaceID)
}
