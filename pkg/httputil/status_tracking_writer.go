package httputil

import (
	"net/http"
)

// StatusTrackingWriter tracks status code written to ResponseWriter instance.
type StatusTrackingWriter struct {
	statusCode *int

	http.ResponseWriter
}

// NewStatusTrackingWriter returns new StatusTrackingWriter.
func NewStatusTrackingWriter(w http.ResponseWriter) *StatusTrackingWriter {
	return &StatusTrackingWriter{
		ResponseWriter: w,
	}
}

func (w *StatusTrackingWriter) recordStatusCodeOnce(statusCode int) {
	if w.statusCode == nil {
		w.statusCode = &statusCode
	}
}

// GetStatusCode returns recorded status code. Returns nil if no status code was recorded.
func (w *StatusTrackingWriter) GetStatusCode() *int {
	return w.statusCode
}

// WriteHeader records statusCode and calls underlying WriteHeader.
func (w *StatusTrackingWriter) WriteHeader(statusCode int) {
	w.recordStatusCodeOnce(statusCode)
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write records statusCode and calls underlying Write.
func (w *StatusTrackingWriter) Write(buf []byte) (int, error) {
	w.recordStatusCodeOnce(http.StatusOK)
	return w.ResponseWriter.Write(buf)
}
