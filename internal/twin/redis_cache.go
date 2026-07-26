package twin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	client *redis.Client
	prefix string
}

func NewRedisCache(client *redis.Client, prefix string) (*RedisCache, error) {
	if client == nil {
		return nil, errors.New("redis client is required")
	}
	if prefix == "" {
		prefix = "opendroneops"
	}
	return &RedisCache{client: client, prefix: prefix}, nil
}

func (c *RedisCache) SetLatest(ctx context.Context, state domain.DeviceState, ttl time.Duration) error {
	value, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal latest state: %w", err)
	}
	if err := c.client.Set(ctx, c.key(state.WorkspaceID, state.DeviceID), value, ttl).Err(); err != nil {
		return fmt.Errorf("set latest state cache: %w", err)
	}
	return nil
}

func (c *RedisCache) GetLatest(ctx context.Context, workspaceID, deviceID domain.ID) (domain.DeviceState, error) {
	value, err := c.client.Get(ctx, c.key(workspaceID, deviceID)).Bytes()
	if err != nil {
		return domain.DeviceState{}, err
	}
	var state domain.DeviceState
	if err := json.Unmarshal(value, &state); err != nil {
		return domain.DeviceState{}, fmt.Errorf("unmarshal latest state cache: %w", err)
	}
	return state, nil
}

func (c *RedisCache) key(workspaceID, deviceID domain.ID) string {
	return fmt.Sprintf("%s:twin:%s:%s:latest", c.prefix, workspaceID, deviceID)
}
