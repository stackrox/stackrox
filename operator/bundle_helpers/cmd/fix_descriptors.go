package cmd

import (
	"bytes"
	"errors"
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

	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("failed to read stdin: %w", err)
	}

	out, err := fixSpecDescriptorOrderBytes(in)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(out)
	return err
}

// fixSpecDescriptorOrderBytes fixes the ordering of specDescriptors in CSV YAML bytes
func fixSpecDescriptorOrderBytes(in []byte) ([]byte, error) {
	var csvDoc map[string]any
	if err := yaml.Unmarshal(in, &csvDoc); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := descriptor.FixCSVDescriptorsMap(csvDoc); err != nil {
		return nil, fmt.Errorf("failed to fix descriptors: %w", err)
	}

	var buf bytes.Buffer
	if err := encodeAndNormalizeYAML(csvDoc, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// encodeAndNormalizeYAML encodes a document to YAML and normalizes it via Python
func encodeAndNormalizeYAML(doc any, w io.Writer) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}
	return normalizeYAMLOutput(buf.Bytes(), w)
}

// normalizeYAMLOutput pipes YAML through the Python normalizer to match PyYAML formatting.
// This handles formatting quirks (quote styles, line wrapping, etc.) while keeping
// all business logic in Go.
func normalizeYAMLOutput(goYAML []byte, w io.Writer) error {
	// Find yaml-normalizer.py: try current directory first (when run from bundle_helpers/),
	// then try bundle_helpers/ subdirectory (when run from operator/)
	normalizerPath := "yaml-normalizer.py"
	if _, err := os.Stat(normalizerPath); err != nil {
		normalizerPath = filepath.Join("bundle_helpers", "yaml-normalizer.py")
		if _, err := os.Stat(normalizerPath); err != nil {
			return errors.New("yaml-normalizer.py not found in current directory or bundle_helpers/ subdirectory")
		}
	}

	cmd := exec.Command(normalizerPath)
	cmd.Stdin = bytes.NewReader(goYAML)
	cmd.Stdout = w
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to normalize YAML: %w", err)
	}

	return nil
}
