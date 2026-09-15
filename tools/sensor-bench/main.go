package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"
)

func main() {
	configPath := flag.String("config", "", "path to scenario YAML config (required)")
	outputDir := flag.String("output", "", "output directory for profiles and metrics (required)")
	cpuProfileSecs := flag.Int("cpu-profile-seconds", 0, "CPU profile duration in seconds (0 to use wall-clock of scenario)")
	flag.Parse()

	if *configPath == "" || *outputDir == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Loading config: %v", err)
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Creating output dir: %v", err)
	}

	log.Println("Starting sensor harness...")
	harness, err := NewHarness(cfg)
	if err != nil {
		log.Fatalf("Creating harness: %v", err)
	}
	defer harness.Stop()

	log.Println("Setting up completion checker...")
	checker, err := NewCompletionChecker(cfg, harness.FakeCentral)
	if err != nil {
		log.Fatalf("Creating completion checker: %v", err)
	}

	// Wait briefly for sensor to fully initialize (metrics/pprof servers)
	time.Sleep(1 * time.Second)

	cpuFile, err := os.Create(filepath.Join(*outputDir, "cpu.pb.gz"))
	if err != nil {
		log.Fatalf("Creating CPU profile file: %v", err)
	}
	if err := pprof.StartCPUProfile(cpuFile); err != nil {
		log.Fatalf("Starting CPU profile: %v", err)
	}

	log.Printf("Injecting workload: %d deployments, %d roles, %d services across %d namespaces...",
		cfg.Workload.Deployments.Count,
		cfg.Workload.Roles.Count,
		cfg.Workload.Services.Count,
		cfg.Workload.Namespaces)

	startTime := time.Now()
	if err := RunScenario(context.Background(), harness, cfg); err != nil {
		log.Fatalf("Running scenario: %v", err)
	}
	log.Println("Injection complete. Waiting for completion conditions...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Completion.Timeout)
	defer cancel()

	select {
	case <-checker.Done.Done():
		log.Println("All completion conditions met.")
	case <-ctx.Done():
		log.Println("WARNING: timeout reached before all conditions were met")
		for _, c := range checker.conditions {
			log.Printf("  %s", c)
		}
	}
	endTime := time.Now()

	if *cpuProfileSecs > 0 {
		log.Printf("Collecting CPU profile for %d additional seconds...", *cpuProfileSecs)
		time.Sleep(time.Duration(*cpuProfileSecs) * time.Second)
	}
	pprof.StopCPUProfile()
	cpuFile.Close()

	log.Println("Collecting profiles and metrics...")
	if err := collectProfiles(*outputDir); err != nil {
		log.Printf("WARNING: collecting profiles: %v", err)
	}
	if err := collectMetrics(*outputDir); err != nil {
		log.Printf("WARNING: collecting metrics: %v", err)
	}
	if err := writeRunMetadata(*outputDir, cfg, startTime, endTime); err != nil {
		log.Printf("WARNING: writing run metadata: %v", err)
	}

	log.Printf("Done. Results in %s (wall time: %s)", *outputDir, endTime.Sub(startTime))
}

func printConditionStatus(checker *CompletionChecker) {
	for _, c := range checker.conditions {
		fmt.Printf("  %s\n", c)
	}
}
