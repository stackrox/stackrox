package logging

import (
	"fmt"
	"io"
)

// Writer wraps an io.Writer and allows to conveniently annotate the written output with fields, similar to Logger.
type Writer struct {
	writer io.Writer
	fields Fields
}

// NewWriter returns a new writer instance.
func NewWriter(writer io.Writer) *Writer {
	return &Writer{
		writer: writer,
	}
}

// WithFields returns a new writer, with fields derived by updating the fields of the receiver writer.
func (w *Writer) WithFields(newFields Fields) *Writer {
	return &Writer{
		writer: w.writer,
		fields: w.fields.update(newFields),
	}
}

// Write prints the arguments without the log prefixes - if and only if - SetOutput has been previously called
func (w *Writer) Write(args ...interface{}) {
	w.writer.Write([]byte(fmt.Sprint(args...) + w.fields.String()))
}

// Writef prints the formatted arguments without the log prefixes - if and only if - SetOutput has been previously called
func (w *Writer) Writef(format string, args ...interface{}) {
	w.writer.Write([]byte(fmt.Sprintf(format, args...) + w.fields.String()))
}
