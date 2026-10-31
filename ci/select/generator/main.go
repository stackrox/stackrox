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
	if config.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", config.Version)
	}

	if len(config.Jobs) == 0 {
		return fmt.Errorf("no jobs defined")
	}

	// Check for duplicate job names
	jobSet := make(map[string]bool)
	for _, job := range config.Jobs {
		if jobSet[job] {
			return fmt.Errorf("duplicate job name %q", job)
		}
		jobSet[job] = true
	}

	// Check that all jobs have defaults
	for _, job := range config.Jobs {
		if _, ok := config.Defaults[job]; !ok {
			return fmt.Errorf("job %q has no default", job)
		}
	}

	// Validate defaults are "run" or "skip"
	for job, action := range config.Defaults {
		if action != "run" && action != "skip" {
			return fmt.Errorf("job %q has invalid default %q (must be 'run' or 'skip')", job, action)
		}
	}

	// Validate rules
	ruleNames := make(map[string]bool)
	for i, rule := range config.Rules {
		if rule.Name == "" {
			return fmt.Errorf("rule %d has no name", i)
		}

		// Check for duplicate rule names
		if ruleNames[rule.Name] {
			return fmt.Errorf("duplicate rule name %q", rule.Name)
		}
		ruleNames[rule.Name] = true

		// Check that condition has exactly one matcher
		numMatchers := 0
		if rule.When.Label != "" {
			numMatchers++
		}
		if len(rule.When.AnyFileMatches) > 0 {
			numMatchers++
		}
		if len(rule.When.EveryFileMatches) > 0 {
			numMatchers++
		}
		if rule.When.AnyFileChanged {
			numMatchers++
		}

		if numMatchers == 0 {
			return fmt.Errorf("rule %q has no condition", rule.Name)
		}
		if numMatchers > 1 {
			return fmt.Errorf("rule %q has multiple conditions (only one allowed)", rule.Name)
		}

		// Validate regex patterns
		for _, pattern := range rule.When.AnyFileMatches {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("rule %q has invalid regex pattern %q: %w", rule.Name, pattern, err)
			}
		}
		for _, pattern := range rule.When.EveryFileMatches {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("rule %q has invalid regex pattern %q: %w", rule.Name, pattern, err)
			}
		}

		// Check that action has run or skip
		if rule.Action.Run == nil && rule.Action.Skip == nil {
			return fmt.Errorf("rule %q has no action (run or skip)", rule.Name)
		}
	}

	return nil
}
