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

package grpcweb

import (
	"bytes"
	"encoding/binary"
	"net/http"
	"strings"

	"golang.stackrox.io/grpc-http1/internal/sliceutils"
	"golang.stackrox.io/grpc-http1/internal/stringutils"
)

type responseWriter struct {
	w http.ResponseWriter

	// List of trailers that were announced via the `Trailer` header at the time headers were written. Also used to keep
	// track of whether headers were already written (in which case this is non-nil, even if it is the empty slice).
	announcedTrailers []string
}

// NewResponseWriter returns a response writer that transparently transcodes an gRPC HTTP/2 response to a gRPC-Web
// response. It can be used as the response writer in the `ServeHTTP` method of a `grpc.Server`.
// The second return value is a finalization function that takes care of sending the data frame with trailers. It
// *needs* to be called before the response handler exits successfully (the returned error is simply any error of the
// underlying response writer passed through).
func NewResponseWriter(w http.ResponseWriter) (http.ResponseWriter, func() error) {
	rw := &responseWriter{
		w: w,
	}
	return rw, rw.Finalize
}

// Header returns the HTTP Header of the underlying response writer.
func (w *responseWriter) Header() http.Header {
	return w.w.Header()
}

// Flush flushes any data not yet written. In contrast to most `http.ResponseWriter` implementations, it does not send
// headers if no data has been written yet.
func (w *responseWriter) Flush() {
	if w.announcedTrailers == nil {
		return // don't send headers needlessly, otherwise trailer-only responses won't work.
	}

	w.prepareHeadersIfNecessary()
	if flusher, _ := w.w.(http.Flusher); flusher != nil {
		flusher.Flush()
	}
}

// prepareHeadersIfNecessary is called internally on any action that might cause headers to be sent.
func (w *responseWriter) prepareHeadersIfNecessary() {
	if w.announcedTrailers != nil {
		return
	}

	hdr := w.w.Header()
	w.announcedTrailers = sliceutils.StringClone(hdr["Trailer"])
	// Trailers are sent in a data frame, so don't announce trailers as otherwise downstream proxies might get confused.
	hdr.Del("Trailer")

	// "Downgrade" response content type to grpc-web.
	contentType, contentSubtype := stringutils.Split2(hdr.Get("Content-Type"), "+")

	respContentType := "application/grpc-web"
	if contentType == "application/grpc" && contentSubtype != "" {
		respContentType += "+" + contentSubtype
	}

	hdr.Set("Content-Type", respContentType)
	// Any content length that might be set is no longer accurate because of trailers.
	hdr.Del("Content-Length")
}

// WriteHeader sends HTTP headers to the client, along with the given status code.
func (w *responseWriter) WriteHeader(statusCode int) {
	w.prepareHeadersIfNecessary()
	w.w.WriteHeader(statusCode)
}

// Write writes a chunk of data.
func (w *responseWriter) Write(buf []byte) (int, error) {
	w.prepareHeadersIfNecessary()
	return w.w.Write(buf)
}

// Finalize sends trailer data in a data frame. It *needs* to be called
func (w *responseWriter) Finalize() error {
	hdr := w.w.Header()
	var trailers http.Header
	if w.announcedTrailers == nil {
		// Trailer-only response! Send trailers as headers...
		trailers = hdr
		delete(trailers, "Trailer")
	} else {
		trailers = make(http.Header)
		for _, at := range w.announcedTrailers {
			at = http.CanonicalHeaderKey(at)
			trailers[at] = hdr[at]
		}
	}

	for k, vs := range hdr {
		if !strings.HasPrefix(k, http.TrailerPrefix) {
			continue
		}
		trailerName := http.CanonicalHeaderKey(k[len(http.TrailerPrefix):])
		trailers[trailerName] = append(trailers[trailerName], vs...)
		delete(hdr, k)
	}

	if w.announcedTrailers == nil {
		return nil // trailer-only response, don't send data frame.
	}

	var buf bytes.Buffer
	if err := trailers.Write(&buf); err != nil {
		return err // should not happen, only errors if (*bytes.Buffer).Write errors.
	}

	trailerFrameHeader := []byte{trailerMessageFlag, 0, 0, 0, 0}
	binary.BigEndian.PutUint32(trailerFrameHeader[1:], uint32(buf.Len()))
	if _, err := w.w.Write(trailerFrameHeader); err != nil {
		return err
	}
	if _, err := w.w.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}
