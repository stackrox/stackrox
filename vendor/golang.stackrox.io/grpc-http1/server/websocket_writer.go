// Copyright (c) 2020 StackRox Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License

package server

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	"github.com/golang/glog"
	"golang.stackrox.io/grpc-http1/internal/grpcproto"
	"golang.stackrox.io/grpc-http1/internal/sliceutils"
)

// wsResponseWriter is a http.ResponseWriter to be used for WebSocket connections.
// (*wsResponseWriter).Close *must* be called when the struct is no longer needed.
type wsResponseWriter struct {
	writer            io.WriteCloser
	header            http.Header
	headerWritten     bool
	announcedTrailers []string
}

// newWebSocketResponseWriter returns a new WebSocket response writer and its relative io.ReadCloser.
// (*wsResponseWriter).Close *must* be called when the struct is no longer needed to signal
// to the reader that there will be no more messages.
func newWebSocketResponseWriter() (*wsResponseWriter, io.ReadCloser) {
	r, w := io.Pipe()
	rw := &wsResponseWriter{
		writer: w,
		header: make(http.Header),
	}
	return rw, r
}

func (w *wsResponseWriter) Write(p []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.writer.Write(p)
}

func (w *wsResponseWriter) Header() http.Header {
	return w.header
}

func (w *wsResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	if statusCode != http.StatusOK && statusCode != http.StatusUnsupportedMediaType {
		glog.Errorf("gRPC server sending unexpected status code: %d", statusCode)
	}

	hdr := w.header
	w.announcedTrailers = sliceutils.StringClone(hdr["Trailer"])
	// Trailers will be sent un-announced in non-Trailers-only responses.
	hdr.Del("Trailer")

	// Any content length that might be set is no longer accurate because of trailers.
	hdr.Del("Content-Length")

	// Write the response header.
	var buf bytes.Buffer
	_ = hdr.Write(&buf)

	// Ignore errors, as WriteHeader does not seem to handle errors.
	_, _ = w.writer.Write(grpcproto.MakeMessageHeader(grpcproto.MetadataFlags, uint32(buf.Len())))
	_, _ = w.writer.Write(buf.Bytes())

	// Mark down that we have written the headers.
	w.headerWritten = true
}

// Flush is a No-Op since the underlying writer is a io.PipeWriter,
// which does no internal buffering.
func (w *wsResponseWriter) Flush() {}

// Close sends over trailers for normal and Trailer-Only gRPC responses.
func (w *wsResponseWriter) Close() error {
	hdr := w.header
	var trailers http.Header
	if w.announcedTrailers == nil {
		// Trailer-only response. Sending the trailers as normal headers.
		trailers = hdr
		delete(trailers, "Trailer")
	} else {
		trailers = make(http.Header)
		// Get the announced trailers from the header.
		for _, at := range w.announcedTrailers {
			at = http.CanonicalHeaderKey(at)
			trailers[at] = hdr[at]
		}
	}

	// Get any unannounced trailers and set them as normal headers.
	// This is in-case we have a trailers-only response.
	prefixLen := len(http.TrailerPrefix)
	for k, vs := range hdr {
		if !strings.HasPrefix(k, http.TrailerPrefix) {
			continue
		}
		trailerName := http.CanonicalHeaderKey(k[prefixLen:])
		trailers[trailerName] = append(trailers[trailerName], vs...)
		delete(hdr, k)
	}

	// Close the pipe when done, so the reader knows to stop.
	// Ignore close error. The underlying writer is an io.Pipe, so errors should not happen.
	defer w.writer.Close()

	var buf bytes.Buffer
	if err := trailers.Write(&buf); err != nil {
		return err // should not happen, only errors if (*bytes.Buffer).Write errors.
	}

	// Write the trailers.
	if _, err := w.writer.Write(grpcproto.MakeMessageHeader(grpcproto.MetadataFlags, uint32(buf.Len()))); err != nil {
		return err
	}
	if _, err := w.writer.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
