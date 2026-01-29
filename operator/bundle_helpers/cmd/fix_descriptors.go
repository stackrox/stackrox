package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stackrox/rox/operator/bundle_helpers/pkg/descriptor"
	"gopkg.in/yaml.v3"
)

// FixSpecDescriptorOrder fixes the ordering of specDescriptors in a CSV file.
// It reads from stdin and writes to stdout, matching the Python script behavior.
func FixSpecDescriptorOrder(args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Println("Usage: bundle-helper fix-spec-descriptor-order < input.yaml > output.yaml")
		fmt.Println()
		fmt.Println("Fixes the ordering of specDescriptors in a ClusterServiceVersion YAML file.")
		fmt.Println("Ensures parent descriptors appear before their children.")
		return nil
	}

	// Read CSV from stdin
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	// Parse YAML into a map (like Python's yaml.safe_load)
	var csvDoc map[string]interface{}
	if err := yaml.Unmarshal(data, &csvDoc); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Process descriptors
	if err := descriptor.FixCSVDescriptorsMap(csvDoc); err != nil {
		return fmt.Errorf("failed to fix descriptors: %w", err)
	}

	// Encode to YAML using Go's yaml.v3
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(csvDoc); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}

	// Normalize through Python to match PyYAML's exact formatting
	// This is the "escape hatch" mentioned in the migration plan
	return normalizeYAMLOutput(buf.Bytes(), os.Stdout)
}

// normalizeYAMLOutput pipes YAML through the Python normalizer to match PyYAML formatting.
// This handles formatting quirks (quote styles, line wrapping, etc.) while keeping
// all business logic in Go.
func normalizeYAMLOutput(goYAML []byte, w io.Writer) error {
	// Find the yaml-normalizer.py script
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	scriptDir := filepath.Dir(execPath)
	normalizerPath := filepath.Join(scriptDir, "..", "yaml-normalizer.py")

	// If running from source (not installed), try the current directory
	if _, err := os.Stat(normalizerPath); os.IsNotExist(err) {
		wd, _ := os.Getwd()
		normalizerPath = filepath.Join(wd, "yaml-normalizer.py")
	}

	// Run the normalizer
	cmd := exec.Command(normalizerPath)
	cmd.Stdin = bytes.NewReader(goYAML)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to normalize YAML: %w", err)
	}

	return nil
}
