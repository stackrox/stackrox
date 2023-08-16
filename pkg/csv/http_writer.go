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

	errSendHeaders = errox.ServerError.New("failed to send headers")
	errSendBody    = errox.ServerError.New("failed to send body")
)

const (
	// ContentType is the Content-Type HTTP header value.
	ContentType = "text/csv; charset=utf-8"
	// UTF8BOM is the UTF-8 BOM byte sequence.
	UTF8BOM = "\uFEFF"
)

// Row is type to hold a CSV line.
type Row = []string

// Converter is type of a function, that converts a custom record object
// to a CSV row.
type Converter[Record any] func(*Record) Row

type httpWriterImpl[Record any] struct {
	writer      http.ResponseWriter
	csv         *csv.Writer
	filename    string
	converter   Converter[Record]
	csvHeader   Row
	headersSent bool
}

// NewHTTPWriter creates an instance of an HTTP CSV writer.
func NewHTTPWriter[Record any](w http.ResponseWriter, filename string,
	converter Converter[Record], csvHeader Row) *httpWriterImpl[Record] {

	csvWriter := csv.NewWriter(w)
	csvWriter.UseCRLF = true
	return &httpWriterImpl[Record]{
		writer:    w,
		csv:       csvWriter,
		filename:  filename,
		converter: converter,
		csvHeader: csvHeader,
	}
}

func (w *httpWriterImpl[Record]) sendHeaders() error {
	// Set UTF-8 BOM to please Windows CSV editors.
	if _, err := w.writer.Write(([]byte)(UTF8BOM)); err != nil {
		return errSendHeaders.CausedBy(err)
	}
	h := w.writer.Header()
	h.Set("Content-Type", ContentType)
	h.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, w.filename))
	w.headersSent = true
	if err := w.csv.Write(w.csvHeader); err != nil {
		return errSendHeaders.CausedBy(err)
	}
	return nil
}

func (w *httpWriterImpl[Record]) Write(row *Record) error {
	if !w.headersSent {
		if err := w.sendHeaders(); err != nil {
			return w.SetHTTPError(err)
		}
	}
	if err := w.csv.Write(w.converter(row)); err != nil {
		err = errSendBody.CausedBy(err)
		return w.SetHTTPError(err)
	}
	return nil
}

func (w *httpWriterImpl[Record]) SetHTTPError(err error) error {
	if err == nil {
		return nil
	}
	if w.headersSent {
		log.Error("Failed to send CSV data:", err)
		// Too late to change the HTTP headers and status.
		return nil
	}
	http.Error(w.writer, err.Error(), http.StatusInternalServerError)
	return err
}

func (w *httpWriterImpl[Record]) Flush() {
	w.csv.Flush()
	_ = w.SetHTTPError(w.csv.Error())
}
