package printers

import (
	"bytes"
	"encoding/json"
	"io"
)

// JSONPrinterOption is a functional option for the JSONPrinter.
type JSONPrinterOption func(*JSONPrinter)

// WithJSONCompact is a functional option for printing the JSON output in compact format.
// By default, the JSON output will not be in compact format.
func WithJSONCompact(compact bool) JSONPrinterOption {
	return func(p *JSONPrinter) {
		p.compact = compact
	}
}

// WithJSONEscapeHTML is a functional option for escaping HTML characters when printing.
// By default, HTML characters will not be escaped.
func WithJSONEscapeHTML(escapeHTML bool) JSONPrinterOption {
	return func(p *JSONPrinter) {
		p.escapeHTML = escapeHTML
	}
}

// JSONPrinter will print an interface that can be passed to json.Encode.
type JSONPrinter struct {
	compact bool
	// escapeHTML instructs JSON marshaller to replace certain
	// characters with their unicode codepoints. This setting is not expected
	// to be added as a flag to the cobra.Command for user input, it should be
	// decided by the command itself.
	escapeHTML bool
}

// NewJSONPrinter returns a printer with configurations capable of printing output formatted as JSON.
func NewJSONPrinter(options ...JSONPrinterOption) *JSONPrinter {
	printer := &JSONPrinter{}
	for _, opt := range options {
		opt(printer)
	}

	return printer
}

// Print will print the specified JSON object to the io.Writer.
// It will return an error if there is an issue with writing to the io.Writer or the passed JSON object.
func (j *JSONPrinter) Print(jsonObject interface{}, out io.Writer) error {
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
