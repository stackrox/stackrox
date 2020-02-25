package metrics

import (
	"net/http"
)

type monitoringResponseWriter struct {
	statusCode *int

	http.ResponseWriter
}

func newMonitoringResponseWriter(w http.ResponseWriter) *monitoringResponseWriter {
	return &monitoringResponseWriter{
		ResponseWriter: w,
	}
}

func (w *monitoringResponseWriter) recordStatusCodeOnce(statusCode int) {
	if w.statusCode == nil {
		w.statusCode = &statusCode
	}
}

func (w *monitoringResponseWriter) GetStatusCode() *int {
	return w.statusCode
}

func (w *monitoringResponseWriter) WriteHeader(statusCode int) {
	w.recordStatusCodeOnce(statusCode)
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *monitoringResponseWriter) Write(buf []byte) (int, error) {
	w.recordStatusCodeOnce(http.StatusOK)
	return w.ResponseWriter.Write(buf)
}
