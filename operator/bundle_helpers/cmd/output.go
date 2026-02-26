package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

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
