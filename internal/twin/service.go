package twin

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

var ErrStaleState = domain.ErrStaleState

type LatestStateStore interface {
	UpsertLatest(context.Context, domain.DeviceState) (bool, error)
	GetLatest(context.Context, domain.ID, domain.ID) (domain.DeviceState, error)
}

type EventStore interface {
	AppendEvent(context.Context, domain.DeviceEvent) (bool, error)
}

type LatestStateCache interface {
	SetLatest(context.Context, domain.DeviceState, time.Duration) error
	GetLatest(context.Context, domain.ID, domain.ID) (domain.DeviceState, error)
}

type ApplyResult struct {
	Accepted     bool
	CacheUpdated bool
	CacheError   error
}

type Service struct {
	states   LatestStateStore
	events   EventStore
	cache    LatestStateCache
	cacheTTL time.Duration
}

func NewService(states LatestStateStore, events EventStore, cache LatestStateCache, cacheTTL time.Duration) (*Service, error) {
	if states == nil || events == nil {
		return nil, errors.New("twin state and event stores are required")
	}
	if cacheTTL <= 0 {
		cacheTTL = 24 * time.Hour
	}
	return &Service{states: states, events: events, cache: cache, cacheTTL: cacheTTL}, nil
}

func (s *Service) ApplyState(ctx context.Context, state domain.DeviceState) (ApplyResult, error) {
	if err := validateState(state); err != nil {
		return ApplyResult{}, err
	}
	accepted, err := s.states.UpsertLatest(ctx, state)
	if err != nil {
		return ApplyResult{}, fmt.Errorf("persist latest device state: %w", err)
	}
	if !accepted {
		return ApplyResult{}, ErrStaleState
	}
	result := ApplyResult{Accepted: true}
	if s.cache != nil {
		if err := s.cache.SetLatest(ctx, state, s.cacheTTL); err != nil {
			result.CacheError = fmt.Errorf("update derived state cache: %w", err)
			return result, nil
		}
		result.CacheUpdated = true
	}
	return result, nil
}

func (s *Service) RecordEvent(ctx context.Context, event domain.DeviceEvent) (bool, error) {
	if err := validateEvent(event); err != nil {
		return false, err
	}
	accepted, err := s.events.AppendEvent(ctx, event)
	if err != nil {
		return false, fmt.Errorf("persist device event: %w", err)
	}
	return accepted, nil
}

func (s *Service) RebuildCache(ctx context.Context, workspaceID, deviceID domain.ID) error {
	if s.cache == nil {
		return nil
	}
	state, err := s.states.GetLatest(ctx, workspaceID, deviceID)
	if err != nil {
		return fmt.Errorf("load latest state for cache rebuild: %w", err)
	}
	if err := s.cache.SetLatest(ctx, state, s.cacheTTL); err != nil {
		return fmt.Errorf("rebuild derived state cache: %w", err)
	}
	return nil
}

func validateState(state domain.DeviceState) error {
	if state.DeviceID == "" || state.WorkspaceID == "" || state.StateVersion < 1 || state.ServerTime.IsZero() {
		return errors.New("device state requires workspace, device, positive version, and server time")
	}
	return nil
}

func validateEvent(event domain.DeviceEvent) error {
	if event.EventID == "" || event.WorkspaceID == "" || event.EventType == "" || event.ReceivedAt.IsZero() {
		return errors.New("device event requires event id, workspace, event type, and received time")
	}
	return nil
}
