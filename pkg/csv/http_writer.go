package csv

import (
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	// CSVContentType is the Content-Type HTTP header value.
	CSVContentType = "text/csv; charset=utf-8"
	// UTF8BOM is the UTF-8 BOM byte sequence.
	UTF8BOM = "\uFEFF"
)

type rowConverter[Row any] func(*Row) []string

type httpCSVWriter[Row any] struct {
	writer      http.ResponseWriter
	csv         *csv.Writer
	filename    string
	converter   rowConverter[Row]
	csvHeader   []string
	headersSent bool
}

// NewHTTPCSVWriter creates an instance of an HTTP CSV writer.
func NewHTTPCSVWriter[Row any](w http.ResponseWriter, filename string,
	converter rowConverter[Row], csvHeader []string) *httpCSVWriter[Row] {

	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true
	return &httpCSVWriter[Row]{
		writer:    w,
		csv:       csvWriter,
		filename:  filename,
		converter: converter,
		csvHeader: csvHeader,
	}
}

func (w *httpCSVWriter[Row]) sendHeaders() error {
	// Set UTF-8 BOM to please Windows CSV editors.
	if _, err := w.writer.Write(([]byte)(UTF8BOM)); err != nil {
		return errox.ServerError.CausedBy(err)
	}
	h := w.writer.Header()
	h.Set("Content-Type", CSVContentType)
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, w.filename))
	w.headersSent = true
	if err := w.csv.Write(w.csvHeader); err != nil {
		return errox.ServerError.CausedBy(err)
	}
	return nil
}

func (w *httpCSVWriter[Row]) Write(row *Row) error {
	if !w.headersSent {
		if err := w.sendHeaders(); err != nil {
			return w.SetHTTPError(err)
		}
	}
	if err := w.csv.Write(w.converter(row)); err != nil {
		return w.SetHTTPError(err)
	}
	return nil
}

func (w *httpCSVWriter[Row]) SetHTTPError(err error) error {
	if err == nil {
		return nil
	}
	if w.headersSent {
		log.Error("Failed to send CSV data:", err)
		// Too late to change the HTTP headers and status.
		return nil
	}
	err = errox.ServerError.CausedBy(err)
	http.Error(w.writer, err.Error(), http.StatusInternalServerError)
	return err
}

func (w *httpCSVWriter[Row]) Flush() {
	w.csv.Flush()
	_ = w.SetHTTPError(w.csv.Error())
}
