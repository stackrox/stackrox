package httputil

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type responseHeaderContextKey struct{}

// RESTHandler wraps a function implementing a REST endpoint as an http.Handler. The function receives the incoming HTTP
// request, and returns a response object or an error. In case of a non-nil error, this error is written in JSON/gRPC
// style via `WriteError` (i.e., you can return a plain Golang error (equivalent to status code 500), a gRPC-style
// status, or an `HTTPError`). Otherwise, the returned object is written as JSON (using `jsonpb` if it is a
// `proto.Message`, and `encoding/json` otherwise).
// If you need to mutate response headers, these can be accessed by calling ResponseHeaderFromContext.
// Any errors that occur writing to the response body are simply logged.
func RESTHandler(endpointFunc func(*http.Request) (interface{}, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		reqCtx := context.WithValue(req.Context(), responseHeaderContextKey{}, w.Header())
		resp, err := endpointFunc(req.WithContext(reqCtx))
		if err != nil {
			WriteError(w, err)
			return
		}
		if resp == nil {
			_, err = fmt.Fprint(w, "{}")
		} else if protoMsg, _ := resp.(proto.Message); protoMsg != nil {
			err = (&jsonpb.Marshaler{}).Marshal(w, protoMsg)
		} else {
			err = json.NewEncoder(w).Encode(resp)
		}

		if err != nil {
			log.Errorf("Failed to send response from REST handler: %v", err)
		}
	}
}

// ResponseHeaderFromContext returns the (mutable) response header from a given context. This only works from within a
// REST endpoint function wrapped with RESTHandler, but is guaranteed to always return a non-nil result in this case.
func ResponseHeaderFromContext(ctx context.Context) http.Header {
	hdr, _ := ctx.Value(responseHeaderContextKey{}).(http.Header)
	return hdr
}
