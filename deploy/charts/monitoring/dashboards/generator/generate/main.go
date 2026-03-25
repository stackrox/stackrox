// Package main provides a CLI tool to generate Grafana dashboard JSON files
// from Go dashboard definitions.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	// Dashboard generation calls will be added here per task.
	fmt.Println("No dashboards defined yet. Dashboard generation will be added in subsequent tasks.")
}
