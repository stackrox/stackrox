package csv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"sort"
)

// Header represents a CSV's header line.
type Header []string

// Value represents a CSV's value (non-header row).
type Value []string

// Writer is the interface for something that writes to CSV files.
type Writer interface {
	AddValue(value Value)
	Write(w http.ResponseWriter, filename string)
}

// GenericWriter is a generic CSV Writer.
type GenericWriter struct {
	header Header
	values []Value
	// If set true, will sort before writing out
	// Sorting is done lexicographically, giving preference to earlier columns
	sort bool
}

// NewGenericWriter creates a new CSV Writer using the given header.
func NewGenericWriter(header Header, sort bool) *GenericWriter {
	return &GenericWriter{header: header, sort: sort}
}

// AddValue adds a CSV value (row) to the CSV file.
func (c *GenericWriter) AddValue(value Value) {
	c.values = append(c.values, value)
}

// IsEmpty returns true if there are no values.
func (c *GenericWriter) IsEmpty() bool {
	return len(c.values) == 0
}

// WriteBytes writes out csv header and values to the provided buffer
func (c *GenericWriter) WriteBytes(buf *bytes.Buffer) error {
	cw := csv.NewWriter(buf)
	cw.UseCRLF = true
	_ = cw.Write(c.header)
	for _, v := range c.values {
		if err := cw.Write(v); err != nil {
			return err
		}
	}
	cw.Flush()
	return nil
}

// Write writes back the CSV file contents into the http.ResponseWriter.
func (c *GenericWriter) Write(w http.ResponseWriter, filename string) {
	w.Header().Set("Content-Type", `text/csv; charset="utf-8"`)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.csv"`, filename))
	w.WriteHeader(http.StatusOK)

	if c.sort {
		sort.Slice(c.values, func(i, j int) bool {
			first, second := c.values[i], c.values[j]
			for len(first) > 0 {
				// first has more values, so greater
				if len(second) == 0 {
					return false
				}
				if first[0] < second[0] {
					return true
				}
				if first[0] > second[0] {
					return false
				}
				first = first[1:]
				second = second[1:]
			}
			// second has more values, so first is lesser
			return len(second) > 0
		})
	}

	_, _ = w.Write([]byte("\uFEFF")) // UTF-8 BOM.
	cw := csv.NewWriter(w)
	cw.UseCRLF = true
	_ = cw.Write(c.header)
	for _, v := range c.values {
		_ = cw.Write(v)
	}
	cw.Flush()
}
