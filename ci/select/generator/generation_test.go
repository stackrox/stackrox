package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateProwGateScript(t *testing.T) {
	config := &Config{
		Version: 1,
		Jobs:    []string{"go", "sensor-tests", "ui"},
		Defaults: map[string]string{
			"go":           "run",
			"sensor-tests": "skip",
			"ui":           "run",
		},
		Rules: []Rule{
			{
				Name:        "sensor-label",
				Description: "Run sensor tests when area/sensor label present",
				When: Condition{
					Label: "area/sensor",
				},
				Action: Action{
					Run: []string{"sensor-tests"},
				},
			},
			{
				Name:        "go-changes",
				Description: "Go file changes need Go tests",
				When: Condition{
					AnyFileMatches: []string{`\.go$`},
				},
				Action: Action{
					Run: []string{"go"},
				},
			},
			{
				Name:        "docs-only",
				Description: "Docs-only changes can skip tests",
				When: Condition{
					EveryFileMatches: []string{`\.md$`, `^docs/`},
				},
				Action: Action{
					Skip: []string{"go", "ui"},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "prow-gate.sh")

	if err := generateProwGateScript(config, outputPath); err != nil {
		t.Fatalf("generateProwGateScript() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// Read generated script
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated script: %v", err)
	}

	script := string(content)

	// Verify header
	if !strings.Contains(script, "#!/usr/bin/env bash") {
		t.Error("missing bash shebang")
	}
	if !strings.Contains(script, "GENERATED FILE - DO NOT EDIT") {
		t.Error("missing generated file warning")
	}

	// Verify main function exists
	if !strings.Contains(script, "ci_should_run_job()") {
		t.Error("missing ci_should_run_job() function")
	}

	// Verify helper functions exist
	requiredFunctions := []string{
		"get_changed_files()",
		"get_pr_labels()",
		"has_test_override()",
		"has_label()",
		"any_file_matches()",
		"every_file_matches()",
	}
	for _, fn := range requiredFunctions {
		if !strings.Contains(script, fn) {
			t.Errorf("missing required function: %s", fn)
		}
	}

	// Verify job defaults exist in case statement
	if !strings.Contains(script, "go)") {
		t.Error("missing go in defaults case statement")
	}
	if !strings.Contains(script, "sensor-tests)") {
		t.Error("missing sensor-tests in defaults case statement")
	}
	if !strings.Contains(script, "ui)") {
		t.Error("missing ui in defaults case statement")
	}

	// Verify unknown job handling
	if !strings.Contains(script, "Warning: Unknown job") {
		t.Error("missing unknown job warning")
	}

	// Verify script is executable
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("generated script is not executable")
	}
}

// TestGeneratedBashSyntax verifies generated bash is syntactically valid
func TestGeneratedBashSyntax(t *testing.T) {
	config := &Config{
		Version: 1,
		Jobs:    []string{"go", "sensor-tests"},
		Defaults: map[string]string{
			"go":           "run",
			"sensor-tests": "skip",
		},
		Rules: []Rule{
			{
				Name: "go-changes",
				When: Condition{
					AnyFileMatches: []string{`\.go$`},
				},
				Action: Action{
					Run: []string{"go"},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "gate.sh")

	if err := generateProwGateScript(config, scriptPath); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	// Verify bash syntax is valid
	cmd := exec.Command("bash", "-n", scriptPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated script has syntax errors: %v\nOutput: %s", err, output)
	}

	// Verify script can be sourced without errors
	testScript := filepath.Join(tmpDir, "source_test.sh")
	if err := os.WriteFile(testScript, []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail
source "%s"
echo "OK"
`, scriptPath)), 0755); err != nil {
		t.Fatalf("failed to write test script: %v", err)
	}

	cmd = exec.Command("bash", testScript)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("script failed to source: %v\nOutput: %s", err, output)
	}
}

func TestGenerateGitHubActionsWorkflow(t *testing.T) {
	config := &Config{
		Version: 1,
		Jobs:    []string{"go", "ui"},
		Defaults: map[string]string{
			"go": "run",
			"ui": "run",
		},
		Rules: []Rule{
			{
				Name: "go-changes",
				When: Condition{
					AnyFileMatches: []string{`\.go$`},
				},
				Action: Action{
					Run: []string{"go"},
				},
			},
			{
				Name: "ui-label",
				When: Condition{
					Label: "area/ui",
				},
				Action: Action{
					Run: []string{"ui"},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "detect-changes.yml")

	if err := generateGitHubActionsWorkflow(config, outputPath); err != nil {
		t.Fatalf("generateGitHubActionsWorkflow() error = %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read generated workflow: %v", err)
	}

	workflow := string(content)

	// Verify YAML structure
	if !strings.Contains(workflow, "name: Detect changed files") {
		t.Error("missing workflow name")
	}
	if !strings.Contains(workflow, "GENERATED FILE - DO NOT EDIT") {
		t.Error("missing generated file warning")
	}

	// Verify outputs are defined
	expectedOutputs := []string{"files", "ui_only"}
	for _, output := range expectedOutputs {
		if !strings.Contains(workflow, output+":") {
			t.Errorf("missing output: %s", output)
		}
	}

	// Verify domain flags are created from rules
	// Rule "go-changes" should create domain flag
	if !strings.Contains(workflow, "go_changes") {
		t.Error("missing go_changes domain flag from go-changes rule")
	}
	// Rule "ui-label" should create domain flag
	if !strings.Contains(workflow, "ui_label") {
		t.Error("missing ui_label domain flag from ui-label rule")
	}
}

func TestClashResolutionLogic(t *testing.T) {
	// Test that generated script has clash resolution logic
	config := &Config{
		Version: 1,
		Jobs:    []string{"go"},
		Defaults: map[string]string{
			"go": "run",
		},
		Rules: []Rule{
			{
				Name: "run-rule",
				When: Condition{
					AnyFileMatches: []string{`\.go$`},
				},
				Action: Action{
					Run: []string{"go"},
				},
			},
			{
				Name: "skip-rule",
				When: Condition{
					EveryFileMatches: []string{`\.md$`},
				},
				Action: Action{
					Skip: []string{"go"},
				},
			},
		},
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "gate.sh")

	if err := generateProwGateScript(config, scriptPath); err != nil {
		t.Fatalf("generation failed: %v", err)
	}

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read script: %v", err)
	}

	script := string(content)

	// Verify clash resolution logic exists
	if !strings.Contains(script, "vote_run=") {
		t.Error("missing vote_run variable")
	}
	if !strings.Contains(script, "vote_skip=") {
		t.Error("missing vote_skip variable")
	}
	if !strings.Contains(script, "run wins") {
		t.Error("missing clash resolution comment")
	}

	// Verify rules are evaluated (checking for pattern matching logic)
	if !strings.Contains(script, "any_file_matches") && !strings.Contains(script, "every_file_matches") {
		t.Error("missing file matching logic for rules")
	}

	// Verify default case handling
	if !strings.Contains(script, "go)") {
		t.Error("missing go job in defaults")
	}
}
