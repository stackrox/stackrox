// Package csv implements a CSV stream writer, which is a convenient wrapper
// over encoding/csv.
//
// The convinience is in the lazy header write, which happens only after the
// first AddRow() is called. This allows for using this implementation in the
// HTTP request handlers, where the header has to be written only after some
// data are available from a storage.
//
// The implementation uses generic parameter Row, which has to be of a struct
// type. Default header builder and record converter use reflection.
// Struct field tag 'csv' can be used to override the automatically computed
// header name.
package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	utf8BOM = ([]byte)("\uFEFF") // to please Windows CSV editors.

	log = logging.LoggerForModule()
)

// StreamWriter interface of a CSV stream writer.
type StreamWriter[Row any] interface {
	AddRow(r *Row) error
	Flush() error
}

// RowConverterFunc converts a Row object to a slice of strings.
type RowConverterFunc[Row any] func(*Row) ([]string, error)

type streamWriterImpl[Row any] struct {
	started bool
	output  io.Writer
	csv     *csv.Writer
	options *writerOptions
}

func makeHeader[Row any]() []string {
	var header []string
	t := reflect.TypeOf((*Row)(nil)).Elem()
	for _, f := range reflect.VisibleFields(t) {
		if tag, ok := f.Tag.Lookup("csv"); ok && tag != "" {
			header = append(header, tag)
		} else {
			header = append(header, f.Name)
		}
	}
	return header
}

func makeRowConverter[Row any]() RowConverterFunc[Row] {
	t := reflect.TypeOf((*Row)(nil)).Elem()
	width := t.NumField()
	row := make([]string, 0, width)
	return func(r *Row) ([]string, error) {
		element := reflect.ValueOf(r).Elem()
		row = row[0:0]
		for i := 0; i < width; i++ {
			row = append(row, fmt.Sprint(element.Field(i).Interface()))
		}
		return row, nil
	}
}

// NewStreamWriter returns an initialized CSV stream writer. No data will be
// written to the output by this function.
func NewStreamWriter[Row any](output io.Writer, opts ...Option) StreamWriter[Row] {
	var options writerOptions
	for _, o := range opts {
		o(&options)
	}
	if options.rowConverter == nil {
		options.rowConverter = makeRowConverter[Row]()
	}
	if options.header == nil {
		options.header = makeHeader[Row]()
	}
	return &streamWriterImpl[Row]{
		options: &options,
		output:  output,
	}
}

func (w *streamWriterImpl[Row]) start() error {
	w.started = true
	if w.options.withBOM {
		if n, err := w.output.Write(utf8BOM); err != nil || n != len(utf8BOM) {
			return errors.WithMessage(err, "failed to write BOM header")
		}
	}
	w.csv = csv.NewWriter(w.output)
	w.csv.UseCRLF = w.options.withCRLF
	if w.options.delimiter != 0 {
		w.csv.Comma = w.options.delimiter
	}
	if len(w.options.header) == 0 {
		return nil
	}
	if err := w.csv.Write(w.options.header); err != nil {
		return errors.WithMessage(err, "failed to write CSV header")
	}
	return nil
}

func (w *streamWriterImpl[Row]) AddRow(r *Row) error {
	if !w.started {
		if err := w.start(); err != nil {
			return err
		}
	}
	record, err := w.options.rowConverter.(RowConverterFunc[Row])(r)
	if err != nil {
		return errors.WithMessage(err, "failed to convert CSV record")
	}
	if err := w.csv.Write(record); err != nil {
		return errors.WithMessage(err, "failed to write CSV record")
	}
	return nil
}

func (w *streamWriterImpl[Row]) Flush() error {
	if !w.started {
		if err := w.start(); err != nil {
			return err
		}
	}
	w.csv.Flush()
	return errors.WithMessage(w.csv.Error(), "failed to flush CSV buffer")
}
