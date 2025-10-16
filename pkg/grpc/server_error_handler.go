package grpc

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	apiCommon "github.com/stackrox/rox/generated/api/common"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
)

// We are introducing wrapper struct to intercept and ignore calls on Write
// function, because Write will be called with correct data within our
// errorHandler.
type responseWriterWrapper struct {
	http.ResponseWriter
}

func (rw *responseWriterWrapper) Write(b []byte) (int, error) {
	return 0, nil
}

func (rw *responseWriterWrapper) WriteHeader(statusCode int) {
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriterWrapper) Header() http.Header {
	return rw.ResponseWriter.Header()
}

// With gRCP Gateway V2, returned payload for errors does not contain "error"
// field anymore. This is redundant with "message" field and it was removed.
// This change would be breaking compatibility change for our API. To keep API
// backward compatible we need to keep this field in returned payload.
func errorHandler(ctx context.Context, serv *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	wWrapper := &responseWriterWrapper{w}
	runtime.DefaultHTTPErrorHandler(ctx, serv, marshaler, wWrapper, r, err)

	protoStatus := status.Convert(err).Proto()
	extendedStatus := &apiCommon.ExtendedRpcStatus{}
	extendedStatus.SetCode(protoStatus.GetCode())
	extendedStatus.SetMessage(protoStatus.GetMessage())
	extendedStatus.SetDetails(protoStatus.GetDetails())
	extendedStatus.SetError(protoStatus.GetMessage())

	buf, _ := marshaler.Marshal(extendedStatus)
	if _, err := w.Write(buf); err != nil {
		grpclog.Infof("Failed to write response: %v", err)
	}
}
