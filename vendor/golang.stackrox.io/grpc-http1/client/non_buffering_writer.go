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

package client

import "net/http"

const (
	dontFlushHeadersHeaderKey = "Grpchttp1-Dont-Flush-Headers"
)

type nonBufferingWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

func (w *nonBufferingWriter) flush() {
	w.flusher.Flush()
}

func (w *nonBufferingWriter) WriteHeader(statusCode int) {
	dontFlushHeaders := w.Header().Get(dontFlushHeadersHeaderKey) == "true"
	w.Header().Del(dontFlushHeadersHeaderKey)
	w.ResponseWriter.WriteHeader(statusCode)

	if dontFlushHeaders {
		return
	}
	// Only flush headers for chunked/non-empty response. Otherwise, an empty, header-only response will be translated
	// into a response with headers and a single empty data frame indicating the end of the response body. This will
	// cause the gRPC client to complain about lack of trailers, which are only allowed to be omitted if the initial
	// header frame is also marked as the end of the stream.
	w.flush()
}

func (w *nonBufferingWriter) Write(buf []byte) (int, error) {
	if len(buf) == 0 {
		return 0, nil
	}
	n, err := w.ResponseWriter.Write(buf)
	w.flush()
	return n, err
}

func nonBufferingHandler(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if flusher, _ := w.(http.Flusher); flusher != nil {
			w = &nonBufferingWriter{ResponseWriter: w, flusher: flusher}
		}
		handler.ServeHTTP(w, r)
	}
}
