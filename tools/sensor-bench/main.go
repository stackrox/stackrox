package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/sensor/benchmark"
)

func main() {
	scenarioDir := flag.String("scenario", "benchmarks/sensor/scenarios/v0/steady-synthetic-dev", "path to scenario directory")
	outPath := flag.String("out", "scorecard.json", "output scorecard JSON path")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	log.Printf("sensor-bench: scenario=%s out=%s", *scenarioDir, *outPath)
	scorecard, err := benchmark.RunScenario(ctx, *scenarioDir, benchmark.Options{})
	if scorecard != nil {
		data, marshalErr := json.MarshalIndent(scorecard, "", "  ")
		if marshalErr != nil {
			log.Fatalf("marshal scorecard: %v", marshalErr)
		}
		if writeErr := os.WriteFile(*outPath, data, 0o644); writeErr != nil {
			log.Fatalf("write scorecard: %v", writeErr)
		}
	}
	if err != nil {
		log.Fatalf("run scenario: %v", err)
	}
}
