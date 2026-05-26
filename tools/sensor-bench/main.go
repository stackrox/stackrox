package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/stackrox/rox/sensor/benchmark"
)

func main() {
	scenarioDir := flag.String("scenario", "benchmarks/sensor/scenarios/v0/steady-synthetic-dev", "path to scenario directory (run mode)")
	outPath := flag.String("out", "scorecard.json", "output scorecard JSON path (run mode)")
	compareBase := flag.String("compare-base", "", "baseline scorecard JSON (compare mode)")
	compareHead := flag.String("compare-head", "", "candidate scorecard JSON, e.g. PR head (compare mode)")
	compareOut := flag.String("compare-out", "", "write comparison markdown to this file; default stdout (compare mode)")
	flag.Parse()

	if *compareBase != "" || *compareHead != "" {
		if *compareBase == "" || *compareHead == "" {
			log.Fatal("compare mode requires both -compare-base and -compare-head")
		}
		if err := runCompare(*compareBase, *compareHead, *compareOut); err != nil {
			log.Fatal(err)
		}
		return
	}

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

func runCompare(basePath, headPath, outPath string) error {
	baseline, err := benchmark.LoadScorecard(basePath)
	if err != nil {
		return err
	}
	candidate, err := benchmark.LoadScorecard(headPath)
	if err != nil {
		return err
	}

	md, err := benchmark.CompareScorecards(candidate, baseline)
	if err != nil {
		return err
	}

	if outPath == "" {
		fmt.Print(md)
		return nil
	}
	if err := os.WriteFile(outPath, []byte(md), 0o644); err != nil {
		return fmt.Errorf("write comparison: %w", err)
	}
	log.Printf("sensor-bench: wrote comparison to %s", outPath)
	return nil
}
