// Package main provides a CLI tool to generate Grafana dashboard JSON files
// from Go dashboard definitions.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stackrox/rox/deploy/charts/monitoring/dashboards/generator"
)

func writeJSON(dir string, filename string, data map[string]any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", filename, err)
	}
	outPath := filepath.Join(dir, filename)
	if err := os.WriteFile(outPath, b, 0644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}
	fmt.Printf("Generated: %s\n", outPath)
	return nil
}

func main() {
	outDir := "deploy/charts/monitoring/dashboards"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	fmt.Printf("Dashboard output directory: %s\n", outDir)

	// Level 1
	l1 := generator.L1Overview()
	if err := writeJSON(outDir, "stackrox-overview.json", l1.Generate()); err != nil {
		log.Fatal(err)
	}

	// Level 2
	l2 := generator.L2CentralInternals()
	if err := writeJSON(outDir, "central-internals.json", l2.Generate()); err != nil {
		log.Fatal(err)
	}

	// Level 3
	l3si := generator.L3SensorIngestion()
	if err := writeJSON(outDir, "central-sensor-ingestion.json", l3si.Generate()); err != nil {
		log.Fatal(err)
	}

	l3ve := generator.L3VulnEnrichment()
	if err := writeJSON(outDir, "central-vuln-enrichment.json", l3ve.Generate()); err != nil {
		log.Fatal(err)
	}

	// Level 3 Stubs (8 remaining dashboards)
	for _, stub := range generator.L3Stubs() {
		filename := stub.UID + ".json"
		if err := writeJSON(outDir, filename, stub.Generate()); err != nil {
			log.Fatal(err)
		}
	}
}
