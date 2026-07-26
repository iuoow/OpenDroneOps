package twin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
)

func TestApplyStatePersistsBeforeUpdatingCache(t *testing.T) {
	store := &fakeStateStore{accepted: true}
	cache := &fakeCache{}
	service, err := NewService(store, &fakeEventStore{}, cache, time.Hour)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	state := validState(2)
	result, err := service.ApplyState(context.Background(), state)
	if err != nil || !result.Accepted || !result.CacheUpdated {
		t.Fatalf("ApplyState() result=%+v err=%v", result, err)
	}
	if store.calls != 1 || cache.calls != 1 || store.callOrder != "store" || cache.callOrder != "cache" {
		t.Fatalf("PostgreSQL-before-cache ordering failed: store=%+v cache=%+v", store, cache)
	}
}

func TestStaleStateDoesNotTouchCache(t *testing.T) {
	store := &fakeStateStore{accepted: false}
	cache := &fakeCache{}
	service, _ := NewService(store, &fakeEventStore{}, cache, time.Hour)
	_, err := service.ApplyState(context.Background(), validState(1))
	if !errors.Is(err, ErrStaleState) {
		t.Fatalf("expected stale state, got %v", err)
	}
	if cache.calls != 0 {
		t.Fatalf("stale state updated cache %d times", cache.calls)
	}
}

func TestCacheFailureDoesNotUndoPostgresTruth(t *testing.T) {
	store := &fakeStateStore{accepted: true}
	cache := &fakeCache{err: errors.New("redis unavailable")}
	service, _ := NewService(store, &fakeEventStore{}, cache, time.Hour)
	result, err := service.ApplyState(context.Background(), validState(3))
	if err != nil || !result.Accepted || result.CacheError == nil {
		t.Fatalf("cache failure should be reported without rejecting state: result=%+v err=%v", result, err)
	}
}

func TestRecordEventIsIdempotentAndRebuildsCacheFromStore(t *testing.T) {
	store := &fakeStateStore{state: validState(4)}
	events := &fakeEventStore{accepted: true}
	cache := &fakeCache{}
	service, _ := NewService(store, events, cache, time.Hour)
	event := domain.DeviceEvent{EventID: "event-1", WorkspaceID: "workspace-1", EventType: "STATE", ReceivedAt: time.Now().UTC()}
	accepted, err := service.RecordEvent(context.Background(), event)
	if err != nil || !accepted || events.calls != 1 {
		t.Fatalf("RecordEvent() result=%v err=%v", accepted, err)
	}
	if err := service.RebuildCache(context.Background(), "workspace-1", "device-1"); err != nil {
		t.Fatalf("RebuildCache() error = %v", err)
	}
	if cache.calls != 1 || cache.last.StateVersion != 4 {
		t.Fatalf("cache rebuild did not use store truth: %+v", cache)
	}
}

func validState(version int64) domain.DeviceState {
	return domain.DeviceState{
		DeviceID: "device-1", WorkspaceID: "workspace-1", StateVersion: version,
		ServerTime: time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
		Payload:    []byte(`{"online":true}`),
	}
}

type fakeStateStore struct {
	accepted  bool
	calls     int
	state     domain.DeviceState
	callOrder string
}

func (f *fakeStateStore) UpsertLatest(_ context.Context, state domain.DeviceState) (bool, error) {
	f.calls++
	f.callOrder = "store"
	f.state = state
	return f.accepted, nil
}

func (f *fakeStateStore) GetLatest(context.Context, domain.ID, domain.ID) (domain.DeviceState, error) {
	f.callOrder = "store"
	return f.state, nil
}

type fakeEventStore struct {
	accepted bool
	calls    int
}

func (f *fakeEventStore) AppendEvent(context.Context, domain.DeviceEvent) (bool, error) {
	f.calls++
	return f.accepted, nil
}

type fakeCache struct {
	calls     int
	err       error
	last      domain.DeviceState
	callOrder string
}

func (f *fakeCache) SetLatest(_ context.Context, state domain.DeviceState, _ time.Duration) error {
	f.calls++
	f.callOrder = "cache"
	f.last = state
	return f.err
}

func (f *fakeCache) GetLatest(context.Context, domain.ID, domain.ID) (domain.DeviceState, error) {
	return f.last, f.err
}
