package cmd

import (
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func encodeAndNormalizeYAML(doc any, w io.Writer) error {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("failed to encode YAML: %w", err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed to close encoder: %w", err)
	}
	// Extra newline to match the previous output format.
	_, err := fmt.Fprintln(w)
	return err
}
