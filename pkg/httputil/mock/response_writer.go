package mock

import (
	"bytes"
	"net/http"
)

const httpStatusUnset = -1

var _ http.ResponseWriter = (*ResponseWriter)(nil)

// ResponseWriter is stub implementation of http.ResponseWriter.
// It is a basic implementation. It is NOT thread safe.
type ResponseWriter struct {
	data    bytes.Buffer
	code    int
	headers http.Header
	err     error
}

// NewResponseWriter returns implementation of http.ResponseWriter for tests that do not send data over network
// but keep them in memory for investigation.
func NewResponseWriter() *ResponseWriter {
	return NewFailingResponseWriter(nil)
}

// NewFailingResponseWriter returns implementation of http.ResponseWriter that returns error on attempt to write to it
// to emulate e.g. closed connection.
func NewFailingResponseWriter(err error) *ResponseWriter {
	return &ResponseWriter{
		code:    httpStatusUnset,
		headers: make(http.Header),
		err:     err,
	}
}

// Header returns the header map.
func (rw *ResponseWriter) Header() http.Header {
	return rw.headers
}

// Write writes the data to the buffer.
func (rw *ResponseWriter) Write(data []byte) (int, error) {
	if rw.err != nil {
		return 0, rw.err
	}
	if rw.code == httpStatusUnset {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.data.Write(data)
}

// WriteHeader sets status code.
func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.code = statusCode
}

func (rw *ResponseWriter) Code() int {
	return rw.code
}

func (rw *ResponseWriter) Data() []byte {
	return rw.data.Bytes()
}

func (rw *ResponseWriter) DataString() string {
	return rw.data.String()
}

// Reset clears all data from the writer for reuse.
func (rw *ResponseWriter) Reset() {
	rw.data.Reset()
	rw.code = httpStatusUnset
	clear(rw.headers)
}
