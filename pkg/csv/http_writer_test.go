package csv

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
)

type httpWriterMock struct {
	header http.Header
	buf    *bytes.Buffer
	err    error

	status int
}

func (w *httpWriterMock) Header() http.Header    { return w.header }
func (w *httpWriterMock) WriteHeader(status int) { w.status = status }
func (w *httpWriterMock) Write(data []byte) (int, error) {
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
	httpWriterMock := &httpWriterMock{
		header: http.Header{},
		buf:    buf,
	}

	converter := func(r *data) Row { return Row{r.a, r.b} }
	header := Row{"a", "b"}
	w := NewHTTPWriter(httpWriterMock, "filename", converter, header)

	var err error
	assert.Empty(t, httpWriterMock.header)
	for _, m := range metrics {
		if err = w.Write(m); err != nil {
			break
		}
	}
	assert.Equal(t, contentType, httpWriterMock.header["Content-Type"][0])
	assert.NoError(t, err)
	w.Flush()
	assert.Equal(t, "\uFEFFa,b\r\na1,b1\r\na2,b2\r\n", buf.String())
	assert.Equal(t, 0, httpWriterMock.status)
}

func TestHTTPCSVWriterError(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriterMock{
		header: http.Header{},
		buf:    buf,
		err:    errors.New("early error"),
	}

	converter := func(r *data) Row { return Row{r.a, r.b} }
	header := Row{"a", "b"}
	w := NewHTTPWriter(httpWriterMock, "filename", converter, header)

	var err error
	assert.Empty(t, httpWriterMock.header)
	for _, m := range metrics {
		if err = w.Write(m); err != nil {
			break
		}
	}
	assert.Equal(t, "text/plain; charset=utf-8", httpWriterMock.header["Content-Type"][0])
	assert.ErrorIs(t, err, errSendHeaders)
	w.Flush()
	assert.Equal(t, "", buf.String())
	assert.Equal(t, http.StatusInternalServerError, httpWriterMock.status)
}

// TestHTTPCSVWriterSetError ensures SetHTTPError sets appropriate headers and
// HTTP status code
func TestHTTPCSVWriterSetError(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriterMock{
		header: http.Header{},
		buf:    buf,
	}

	converter := func(r *data) Row { return Row{r.a, r.b} }
	header := Row{"a", "b"}
	w := NewHTTPWriter(httpWriterMock, "filename", converter, header)
	_ = w.SetHTTPError(errox.ServerError)
	w.Flush()

	assert.Contains(t, httpWriterMock.header["Content-Type"][0], "text/plain")
	assert.Equal(t, http.StatusInternalServerError, httpWriterMock.status)
}

func TestHTTPCSVWriterLateError(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	httpWriterMock := &httpWriterMock{
		header: http.Header{},
		buf:    buf,
	}

	converter := func(r *data) Row { return Row{r.a, r.b} }
	header := Row{"a", "b"}
	w := NewHTTPWriter(httpWriterMock, "filename", converter, header)

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
	assert.Equal(t, contentType, httpWriterMock.header["Content-Type"][0])
	assert.NoError(t, err)
	w.Flush()
	assert.Equal(t, utf8BOM, buf.String())
	assert.Equal(t, 0, httpWriterMock.status)
}
