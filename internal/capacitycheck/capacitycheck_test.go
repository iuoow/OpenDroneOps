package capacitycheck

import (
	"context"
	"testing"
	"time"
)

func TestRunPassesDeterministicLocalScenarios(t *testing.T) {
	report, err := Run(context.Background(), Config{
		Sessions: 2, Events: 8, Timeout: time.Second, MaxP95Latency: time.Second,
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !report.Passed || len(report.Scenarios) != 4 {
		t.Fatalf("report = %+v", report)
	}
	for _, scenario := range report.Scenarios {
		if !scenario.Passed {
			t.Fatalf("scenario %q failed: %+v", scenario.Name, scenario)
		}
	}
}

func TestRunRejectsInvalidConfiguration(t *testing.T) {
	_, err := Run(context.Background(), Config{Sessions: 0, Events: 1, Timeout: time.Second, MaxP95Latency: time.Second})
	if err == nil {
		t.Fatal("Run() accepted zero sessions")
	}
}
