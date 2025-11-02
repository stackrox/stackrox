package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	outputDir    = flag.String("output", "pkg/postgres/schema", "Output directory for generated schema files")
	verbose      = flag.Bool("verbose", false, "Enable verbose logging")
	discoveryOnly = flag.Bool("discover", false, "Only run discovery without generating files")
	entityFilter = flag.String("entity", "", "Generate only for specific entity (e.g., 'AuthProvider')")
)

func main() {
	flag.Parse()

	if *verbose {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Get the project root directory
	projectRoot, err := findProjectRoot()
	if err != nil {
		log.Fatalf("Failed to find project root: %v", err)
	}

	generator := &SchemaGenerator{
		ProjectRoot:  projectRoot,
		OutputDir:    filepath.Join(projectRoot, *outputDir),
		Verbose:      *verbose,
		EntityFilter: *entityFilter,
	}

	if *discoveryOnly {
		if err := generator.RunDiscovery(); err != nil {
			log.Fatalf("Discovery failed: %v", err)
		}
		fmt.Println("Discovery completed successfully")
		return
	}

	if err := generator.Generate(); err != nil {
		log.Fatalf("Schema generation failed: %v", err)
	}

	fmt.Println("Schema generation completed successfully")
}

// findProjectRoot locates the stackrox project root by looking for go.mod
func findProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir, nil
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("could not find go.mod file")
}