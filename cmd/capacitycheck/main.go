package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/iuoow/OpenDroneOps/internal/capacitycheck"
)

func main() {
	config := capacitycheck.DefaultConfig()
	flag.IntVar(&config.Sessions, "sessions", config.Sessions, "WebSocket sessions for the fan-out scenario")
	flag.IntVar(&config.Events, "events", config.Events, "durable events for the fan-out scenario")
	flag.DurationVar(&config.Timeout, "timeout", config.Timeout, "maximum duration per scenario")
	flag.DurationVar(&config.MaxP95Latency, "max-p95", config.MaxP95Latency, "maximum local WebSocket fan-out p95 latency")
	flag.Parse()

	report, err := capacitycheck.Run(context.Background(), config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "capacity check configuration error:", err)
		os.Exit(2)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "capacity check report error:", err)
		os.Exit(2)
	}
	if !report.Passed {
		os.Exit(1)
	}
}
