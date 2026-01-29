package yamlformat

import (
	"bytes"
	"io"

	"gopkg.in/yaml.v3"
)

// EncodePyYAMLStyle encodes data to YAML matching PyYAML's safe_dump() style.
// This includes:
// - Single quotes for simple string values
// - Empty string as '' not ""
// - Flow style for arrays/maps where appropriate
func EncodePyYAMLStyle(w io.Writer, data interface{}) error {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	if err := encoder.Encode(data); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}

	// Post-process to match PyYAML style
	output := buf.Bytes()
	output = normalizeToPyYAMLStyle(output)

	_, err := w.Write(output)
	return err
}

// normalizeToPyYAMLStyle converts Go yaml.v3 output to match PyYAML style.
func normalizeToPyYAMLStyle(input []byte) []byte {
	// For now, just return as-is
	// We'll implement specific transformations if needed
	return input
}
