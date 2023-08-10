package csv

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type httpWriter struct {
	header http.Header
	buf    *bytes.Buffer
	err    error

	status int
}

func (w *httpWriter) Header() http.Header    { return w.header }
func (w *httpWriter) WriteHeader(status int) { w.status = status }
func (w *httpWriter) Write(data []byte) (int, error) {
	if w.err != nil {
		return 0, w.err
	}
	return w.buf.Write(data)
}

type data struct {
	a string
	b string
}

var metrics = []*data{{a: "a1", b: "b1"}, {a: "a2", b: "b2"}}

func TestHTTPCSVWriter(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriter{
		header: http.Header{},
		buf:    buf,
	}

	rowConverter := func(r *data) []string { return []string{r.a, r.b} }
	header := []string{"a", "b"}
	w := NewHTTPCSVWriter(httpWriterMock, "filename", rowConverter, header)

	var err error
	assert.Empty(t, httpWriterMock.header)
	for _, m := range metrics {
		if err = w.Write(m); err != nil {
			break
		}
	}
	assert.Equal(t, CSVContentType, httpWriterMock.header["Content-Type"][0])
	assert.NoError(t, err)
	w.Flush()
	assert.Equal(t, "\uFEFFa,b\r\na1,b1\r\na2,b2\r\n", buf.String())
	assert.Equal(t, 0, httpWriterMock.status)
}

func TestHTTPCSVWriterError(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriter{
		header: http.Header{},
		buf:    buf,
		err:    errors.New("early error"),
	}

	rowConverter := func(r *data) []string { return []string{r.a, r.b} }
	header := []string{"a", "b"}
	w := NewHTTPCSVWriter(httpWriterMock, "filename", rowConverter, header)

	var err error
	assert.Empty(t, httpWriterMock.header)
	for _, m := range metrics {
		if err = w.Write(m); err != nil {
			break
		}
	}
	assert.Equal(t, "text/plain; charset=utf-8", httpWriterMock.header["Content-Type"][0])
	assert.Error(t, err)
	w.Flush()
	assert.Equal(t, "", buf.String())
	assert.Equal(t, http.StatusInternalServerError, httpWriterMock.status)
}

func TestHTTPCSVWriterLateError(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriter{
		header: http.Header{},
		buf:    buf,
	}

	rowConverter := func(r *data) []string { return []string{r.a, r.b} }
	header := []string{"a", "b"}
	w := NewHTTPCSVWriter(httpWriterMock, "filename", rowConverter, header)

	var err error
	assert.Empty(t, httpWriterMock.header)

	// make csv writer to fail validation. That will be too late to change the
	// status code and the Content-Type header.
	w.csv.Comma = '"'
	for _, m := range metrics {
		if err = w.Write(m); err != nil {
			break
		}
	}
	assert.Equal(t, CSVContentType, httpWriterMock.header["Content-Type"][0])
	assert.NoError(t, err)
	w.Flush()
	assert.Equal(t, UTF8BOM, buf.String())
	assert.Equal(t, 0, httpWriterMock.status)
}
