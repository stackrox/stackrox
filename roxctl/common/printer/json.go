package printer

import (
	"bytes"
	"encoding/json"
	"io"
)

type jsonPrinter struct {
	compact bool
	// escapeHTML instructs JSON marshaller to replace certain
	// characters with their unicode codepoints. This setting is not expected
	// to be added as a flag to the cobra.Command for user input, it should be
	// decided by the command itself.
	escapeHTML bool
}

// newJSONPrinter returns a printer with configurations capable of printing output formatted as JSON
func newJSONPrinter(compact bool, escapeHTML bool) *jsonPrinter {
	return &jsonPrinter{compact: compact, escapeHTML: escapeHTML}
}

func (j *jsonPrinter) Print(jsonObject interface{}, out io.Writer) error {
	switch {
	case j.compact:
		return compactPrint(jsonObject, j.escapeHTML, out)
	default:
		return prettyPrint(jsonObject, j.escapeHTML, out)
	}
}

func prettyPrint(jsonObj interface{}, escapeHTML bool, out io.Writer) error {
	jsonEncoder := createEncoder(escapeHTML, out)
	jsonEncoder.SetIndent("", "  ")
	return jsonEncoder.Encode(jsonObj)
}

func compactPrint(jsonObj interface{}, escapeHTML bool, out io.Writer) error {
	jsonBytes := &bytes.Buffer{}
	enc := createEncoder(escapeHTML, jsonBytes)
	if err := enc.Encode(jsonObj); err != nil {
		return err
	}
	buf := &bytes.Buffer{}
	if err := json.Compact(buf, jsonBytes.Bytes()); err != nil {
		return err
	}
	if _, err := buf.WriteTo(out); err != nil {
		return err
	}
	return nil
}

func createEncoder(escapeHTML bool, out io.Writer) *json.Encoder {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(escapeHTML)
	return enc
}
