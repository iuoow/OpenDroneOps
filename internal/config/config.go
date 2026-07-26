package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv                 string
	HTTPAddr               string
	AdminAddr              string
	PostgresDSN            string
	RedisAddr              string
	RedisPassword          string
	MQTTURL                string
	MQTTClientID           string
	MQTTUsername           string
	MQTTPassword           string
	MQTTSessionExpiry      time.Duration
	MQTTKeepAlive          time.Duration
	RawMessageRetention    time.Duration
	WebSocketSendQueueSize int
	MQTTShardCount         int
	MQTTShardQueueSize     int
}

func Load() (Config, error) {
	cfg := Config{
		AppEnv:                 getenv("APP_ENV", "development"),
		HTTPAddr:               getenv("HTTP_ADDR", ":8080"),
		AdminAddr:              getenv("ADMIN_ADDR", "127.0.0.1:9090"),
		PostgresDSN:            os.Getenv("POSTGRES_DSN"),
		RedisAddr:              getenv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:          os.Getenv("REDIS_PASSWORD"),
		MQTTURL:                getenv("MQTT_URL", "mqtt://localhost:1883"),
		MQTTClientID:           getenv("MQTT_CLIENT_ID", "opendroneops-local"),
		MQTTUsername:           os.Getenv("MQTT_USERNAME"),
		MQTTPassword:           os.Getenv("MQTT_PASSWORD"),
		MQTTSessionExpiry:      seconds("MQTT_SESSION_EXPIRY_SECONDS", 3600),
		MQTTKeepAlive:          seconds("MQTT_KEEP_ALIVE_SECONDS", 30),
		RawMessageRetention:    days("RAW_MESSAGE_RETENTION_DAYS", 7),
		WebSocketSendQueueSize: integer("WS_SEND_QUEUE_SIZE", 256),
		MQTTShardCount:         integer("MQTT_SHARD_COUNT", 32),
		MQTTShardQueueSize:     integer("MQTT_SHARD_QUEUE_SIZE", 1024),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	for name, value := range map[string]string{
		"POSTGRES_DSN":   c.PostgresDSN,
		"MQTT_URL":       c.MQTTURL,
		"MQTT_CLIENT_ID": c.MQTTClientID,
	} {
		if value == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %v", missing)
	}
	if c.WebSocketSendQueueSize < 1 || c.MQTTShardCount < 1 || c.MQTTShardQueueSize < 1 {
		return errors.New("queue and shard sizes must be positive")
	}
	return nil
}

func getenv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func integer(name string, fallback int) int {
	value, err := strconv.Atoi(getenv(name, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func seconds(name string, fallback int) time.Duration {
	return time.Duration(integer(name, fallback)) * time.Second
}

func days(name string, fallback int) time.Duration {
	return time.Duration(integer(name, fallback)) * 24 * time.Hour
}
