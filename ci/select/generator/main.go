package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: generator <config.yaml>")
	}

	configPath := os.Args[1]

	// Read and parse config
	config, err := loadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Validate config
	if err := validateConfig(config); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// Determine output directory (same as config directory)
	outputDir := filepath.Dir(configPath)
	repoRoot := filepath.Dir(outputDir) // ci/ -> repo root

	// Generate GitHub Actions detect-changes workflow
	ghaOutputPath := filepath.Join(repoRoot, ".github", "workflows", "detect-changes-generated.yml")
	if err := generateGitHubActionsWorkflow(config, ghaOutputPath); err != nil {
		log.Fatalf("Failed to generate GitHub Actions workflow: %v", err)
	}
	fmt.Printf("✓ Generated: %s\n", ghaOutputPath)

	// Generate Prow gate script
	prowOutputPath := filepath.Join(outputDir, "prow-native-gate.sh")
	if err := generateProwGateScript(config, prowOutputPath); err != nil {
		log.Fatalf("Failed to generate Prow gate script: %v", err)
	}
	fmt.Printf("✓ Generated: %s\n", prowOutputPath)

	fmt.Println("✓ Generation complete")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	// Check version
	if config.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", config.Version)
	}

	// Check jobs exist
	if len(config.Jobs) == 0 {
		return fmt.Errorf("no jobs defined")
	}

	// Validate each job
	for jobName, job := range config.Jobs {
		// Validate when conditions
		for i, when := range job.When {
			if err := validateWhenCondition(when, fmt.Sprintf("job %q when[%d]", jobName, i)); err != nil {
				return err
			}
		}

		// Validate when_not conditions
		for i, whenNot := range job.WhenNot {
			if err := validateWhenCondition(whenNot, fmt.Sprintf("job %q when_not[%d]", jobName, i)); err != nil {
				return err
			}
		}
	}

	// Validate run_all_tests conditions
	for i, when := range config.RunAllTests.When {
		if err := validateWhenCondition(when, fmt.Sprintf("run_all_tests when[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}

func validateWhenCondition(cond WhenCondition, context string) error {
	// Count how many fields are set
	numSet := 0
	if cond.Always {
		numSet++
	}
	if cond.Label != "" {
		numSet++
	}
	if len(cond.Files) > 0 {
		numSet++
	}

	// Exactly one field must be set
	if numSet == 0 {
		return fmt.Errorf("%s: no condition specified (need always, label, or files)", context)
	}
	if numSet > 1 {
		return fmt.Errorf("%s: multiple conditions specified (only one allowed)", context)
	}

	// Validate regex patterns in files
	for _, pattern := range cond.Files {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("%s: invalid regex pattern %q: %w", context, pattern, err)
		}
	}

	return nil
}
