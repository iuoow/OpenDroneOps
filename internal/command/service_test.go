package command

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/iuoow/OpenDroneOps/internal/domain"
	"github.com/iuoow/OpenDroneOps/internal/protocol/dji"
	"github.com/iuoow/OpenDroneOps/internal/websockethub"
)

func TestCreateIsTransactionalAndIdempotent(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	now := time.Date(2026, 7, 26, 3, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request := validCreateRequest()
	first, created, err := service.Create(context.Background(), request)
	if err != nil || !created || first.Status != domain.CommandPublishPending {
		t.Fatalf("first Create() command=%+v created=%v err=%v", first, created, err)
	}
	second, created, err := service.Create(context.Background(), request)
	if err != nil || created || second.ID != first.ID {
		t.Fatalf("idempotent Create() command=%+v created=%v err=%v", second, created, err)
	}
	if len(repository.commands) != 1 || len(repository.outbox) != 1 || len(repository.events) != 3 || len(repository.audits) != 1 {
		t.Fatalf("transaction bundle counts commands=%d outbox=%d events=%d audits=%d",
			len(repository.commands), len(repository.outbox), len(repository.events), len(repository.audits))
	}
	conflict := request
	conflict.Parameters = []byte(`{"different":true}`)
	if _, _, err := service.Create(context.Background(), conflict); !errors.Is(err, domain.ErrIdempotencyConflict) {
		t.Fatalf("idempotency conflict error=%v", err)
	}
}

func TestConcurrentCreateProducesOneCommand(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	fixed := time.Now().UTC()
	service.now = func() time.Time { return fixed }
	const workers = 12
	var wait sync.WaitGroup
	ids := make(chan domain.ID, workers)
	created := make(chan bool, workers)
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			command, wasCreated, err := service.Create(context.Background(), validCreateRequest())
			if err != nil {
				errs <- err
				return
			}
			ids <- command.ID
			created <- wasCreated
		}()
	}
	wait.Wait()
	close(ids)
	close(created)
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	var first domain.ID
	for id := range ids {
		if first == "" {
			first = id
		}
		if id != first {
			t.Fatalf("idempotent create returned different ids: %s != %s", id, first)
		}
	}
	createdCount := 0
	for value := range created {
		if value {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("created count=%d want=1", createdCount)
	}
}

func TestOutboxPublishSuccessAndFiniteFailure(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	now := time.Date(2026, 7, 26, 4, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	command, _, err := service.Create(context.Background(), validCreateRequest())
	if err != nil {
		t.Fatal(err)
	}
	broker := &fakeBroker{}
	transitionNotifications := 0
	publisher, _ := NewOutboxPublisher(repository, broker, OutboxConfig{
		WorkerID: "worker-1", BatchSize: 1, MaxAttempts: 2,
		InitialBackoff: time.Second, MaxBackoff: time.Second,
		Jitter:           func(time.Duration) time.Duration { return 0 },
		OnCommandChanged: func(domain.Command, time.Time) { transitionNotifications++ },
	})
	publisher.now = func() time.Time { return now }
	count, err := publisher.RunOnce(context.Background())
	if err != nil || count != 1 || broker.calls != 1 || transitionNotifications != 1 {
		t.Fatalf("RunOnce() count=%d calls=%d notifications=%d err=%v", count, broker.calls, transitionNotifications, err)
	}
	if repository.commands[command.ID].Status != domain.CommandPublished {
		t.Fatalf("command status=%s", repository.commands[command.ID].Status)
	}

	failingRepository := NewMemoryRepository()
	failingService, _ := NewService(failingRepository, nil, nil)
	failingService.now = func() time.Time { return now }
	failedCommand, _, _ := failingService.Create(context.Background(), validCreateRequest())
	failingBroker := &fakeBroker{err: errors.New("broker unavailable")}
	failingPublisher, _ := NewOutboxPublisher(failingRepository, failingBroker, OutboxConfig{
		WorkerID: "worker-2", BatchSize: 1, MaxAttempts: 2,
		InitialBackoff: time.Second, MaxBackoff: time.Second,
		Jitter: func(time.Duration) time.Duration { return 0 },
	})
	failingPublisher.now = func() time.Time { return now }
	if _, err := failingPublisher.RunOnce(context.Background()); err != nil {
		t.Fatalf("first failed publish bookkeeping error=%v", err)
	}
	now = now.Add(time.Second)
	if _, err := failingPublisher.RunOnce(context.Background()); err != nil {
		t.Fatalf("final failed publish bookkeeping error=%v", err)
	}
	if failingRepository.commands[failedCommand.ID].Status != domain.CommandFailed || failingBroker.calls != 2 {
		t.Fatalf("finite retry status=%s calls=%d", failingRepository.commands[failedCommand.ID].Status, failingBroker.calls)
	}
}

func TestExpiredLeaseCannotBeCompletedByOldWorker(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	_, _, err := service.Create(context.Background(), validCreateRequest())
	if err != nil {
		t.Fatal(err)
	}
	first, err := repository.LeaseOutbox(context.Background(), "worker-1", 1, now, time.Second)
	if err != nil || len(first) != 1 {
		t.Fatalf("first lease=%+v err=%v", first, err)
	}
	second, err := repository.LeaseOutbox(context.Background(), "worker-2", 1, now.Add(2*time.Second), time.Second)
	if err != nil || len(second) != 1 {
		t.Fatalf("second lease=%+v err=%v", second, err)
	}
	if err := repository.MarkRetry(context.Background(), "worker-1", first[0].Event.ID, now, "late result"); err == nil {
		t.Fatal("expired lease owner updated the outbox")
	}
	if err := repository.MarkRetry(context.Background(), "worker-2", second[0].Event.ID, now.Add(3*time.Second), "retry"); err != nil {
		t.Fatalf("current lease owner failed: %v", err)
	}
}

func TestHighRiskOutboxIsNeverAutomaticallyRetried(t *testing.T) {
	registry, err := NewRegistry([]MethodDefinition{{
		Name: "test_high_risk", Timeout: time.Minute, RiskLevel: domain.RiskHigh, QoS: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewMemoryRepository()
	service, _ := NewService(repository, registry, nil)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	request := validCreateRequest()
	request.Method = "test_high_risk"
	command, _, err := service.Create(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	broker := &fakeBroker{err: errors.New("publish rejected")}
	publisher, _ := NewOutboxPublisher(repository, broker, OutboxConfig{
		WorkerID: "worker-high", MaxAttempts: 5, Jitter: func(time.Duration) time.Duration { return 0 },
	})
	publisher.now = func() time.Time { return now }
	if _, err := publisher.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if broker.calls != 1 || repository.commands[command.ID].Status != domain.CommandFailed {
		t.Fatalf("high-risk retry calls=%d status=%s", broker.calls, repository.commands[command.ID].Status)
	}
}

func TestReplyCorrelationDuplicateAndOrphanHandling(t *testing.T) {
	repository := NewMemoryRepository()
	wsPublisher := &recordingCommandPublisher{}
	service, _ := NewService(repository, nil, wsPublisher)
	now := time.Date(2026, 7, 26, 5, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	command, _, _ := service.Create(context.Background(), validCreateRequest())
	if err := markOnlyOutboxPublished(repository, command.ID, now); err != nil {
		t.Fatal(err)
	}
	payload := []byte(`{"tid":"` + command.DJITID + `","bid":"` + command.DJIBID + `","gateway":"SIM-GW-001","method":"sim_status_refresh","data":{"result":"SUCCEEDED","result_code":0}}`)
	message, err := dji.ParseMessage("thing/product/SIM-GW-001/services_reply", payload)
	if err != nil {
		t.Fatal(err)
	}
	updated, changed, err := service.HandleDJIReply(context.Background(), "workspace-1", message, "", now.Add(time.Second))
	if err != nil || !changed || updated.Status != domain.CommandSucceeded {
		t.Fatalf("HandleDJIReply() command=%+v changed=%v err=%v", updated, changed, err)
	}
	_, changed, err = service.HandleDJIReply(context.Background(), "workspace-1", message, "", now.Add(2*time.Second))
	if err != nil || changed {
		t.Fatalf("duplicate reply changed=%v err=%v", changed, err)
	}

	orphanPayload := []byte(`{"tid":"missing-tid","bid":"missing-bid","gateway":"SIM-GW-001","method":"sim_status_refresh","data":{"result":"FAILED","result_code":500}}`)
	orphan, _ := dji.ParseMessage("thing/product/SIM-GW-001/services_reply", orphanPayload)
	if _, changed, err := service.HandleDJIReply(context.Background(), "workspace-1", orphan, "", now); err != nil || changed {
		t.Fatalf("orphan reply changed=%v err=%v", changed, err)
	}
	if len(repository.orphans) != 1 {
		t.Fatalf("orphan count=%d", len(repository.orphans))
	}
	if _, _, err := service.HandleDJIReply(context.Background(), "workspace-1", orphan, "", now); err != nil {
		t.Fatal(err)
	}
	if len(repository.orphans) != 1 {
		t.Fatalf("duplicate orphan count=%d", len(repository.orphans))
	}
}

func TestReplyCanRepairPublishPendingBeforeDeviceResult(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	now := time.Now().UTC()
	service.now = func() time.Time { return now }
	command, _, _ := service.Create(context.Background(), validCreateRequest())
	payload := []byte(`{"tid":"` + command.DJITID + `","bid":"` + command.DJIBID + `","gateway":"SIM-GW-001","method":"sim_status_refresh","data":{"result":"FAILED","result_code":500}}`)
	message, err := dji.ParseMessage("thing/product/SIM-GW-001/services_reply", payload)
	if err != nil {
		t.Fatal(err)
	}
	updated, changed, err := service.HandleDJIReply(context.Background(), "workspace-1", message, "", now)
	if err != nil || !changed || updated.Status != domain.CommandFailed {
		t.Fatalf("early reply command=%+v changed=%v err=%v", updated, changed, err)
	}
	if len(repository.events) != 5 {
		t.Fatalf("early reply should record publish repair and result events, got %d", len(repository.events))
	}
}

func TestPublishedCommandExpiresToTimeout(t *testing.T) {
	repository := NewMemoryRepository()
	service, _ := NewService(repository, nil, nil)
	now := time.Date(2026, 7, 26, 6, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }
	request := validCreateRequest()
	request.ExpiresAt = now.Add(time.Second)
	command, _, _ := service.Create(context.Background(), request)
	if err := markOnlyOutboxPublished(repository, command.ID, now); err != nil {
		t.Fatal(err)
	}
	now = now.Add(2 * time.Second)
	expired, err := service.Expire(context.Background(), 10)
	if err != nil || len(expired) != 1 || expired[0].Status != domain.CommandTimeout {
		t.Fatalf("Expire() commands=%+v err=%v", expired, err)
	}
	expired, err = service.Expire(context.Background(), 10)
	if err != nil || len(expired) != 0 {
		t.Fatalf("idempotent Expire() commands=%+v err=%v", expired, err)
	}
}

func TestDefaultRegistryRejectsUnapprovedMethods(t *testing.T) {
	service, _ := NewService(NewMemoryRepository(), nil, nil)
	request := validCreateRequest()
	request.Method = "sim_alarm_trigger"
	if _, _, err := service.Create(context.Background(), request); !errors.Is(err, ErrUnknownMethod) {
		t.Fatalf("unapproved method error=%v", err)
	}
}

func validCreateRequest() CreateRequest {
	return CreateRequest{
		WorkspaceID: "workspace-1", TargetDeviceID: "device-1", GatewayDeviceID: "gateway-1",
		GatewaySN: "SIM-GW-001", Method: "sim_status_refresh", Parameters: []byte(`{"refresh":true}`),
		IdempotencyKey: "idem-key-0001", RequestedBy: "operator-1", RequestID: "request-1",
	}
}

func markOnlyOutboxPublished(repository *MemoryRepository, commandID domain.ID, at time.Time) error {
	deliveries, err := repository.LeaseOutbox(context.Background(), "manual-worker", 1, at, time.Minute)
	if err != nil {
		return err
	}
	if len(deliveries) != 1 {
		return errors.New("expected one outbox delivery")
	}
	_, _, err = repository.MarkPublished(context.Background(), "manual-worker", deliveries[0].Event.ID, commandID, at)
	return err
}

type fakeBroker struct {
	calls int
	err   error
}

func (b *fakeBroker) Publish(_ context.Context, _ string, _ []byte, _ byte) error {
	b.calls++
	return b.err
}

type recordingCommandPublisher struct {
	mu     sync.Mutex
	events []websockethub.Event
}

func (p *recordingCommandPublisher) Publish(event websockethub.Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, event)
}
