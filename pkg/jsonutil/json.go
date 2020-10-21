package jsonutil

import (
	"encoding/json"
	"io"
)

// JSONArrayWriter writes out a result as a JSON array in incremental fashion
type JSONArrayWriter struct {
	prependComma bool
	writer       io.Writer
	encoder      *json.Encoder
}

// NewJSONArrayWriter takes in an output writer and creates a new JSONArrayWriter
func NewJSONArrayWriter(writer io.Writer) *JSONArrayWriter {
	return &JSONArrayWriter{
		prependComma: false,
		writer:       writer,
		encoder:      json.NewEncoder(writer),
	}
}

// Init writes a [ to the writer
func (j *JSONArrayWriter) Init() error {
	_, err := j.writer.Write([]byte("["))
	return err
}

// WriteObject writes an interface into JSON and writes it to the writer
func (j *JSONArrayWriter) WriteObject(i interface{}) error {
	if !j.prependComma {
		j.prependComma = true
	} else {
		if _, err := j.writer.Write([]byte(",")); err != nil {
			return err
		}
	}
	return j.encoder.Encode(i)
}

// Finish finishes the array with a ]
func (j *JSONArrayWriter) Finish() error {
	_, err := j.writer.Write([]byte("]"))
	return err
}
