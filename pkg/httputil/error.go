package httputil

import (
	"fmt"
	"net/http"

	"github.com/golang/protobuf/jsonpb"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// HTTPError is an interface for HTTP errors that can be returned from an HTTP handler.
type HTTPError interface {
	error
	HTTPStatus
}

type httpError struct {
	httpStatus
}

func (e httpError) Error() string {
	return e.message
}

// NewError returns a new HTTP error
func NewError(statusCode int, message string) HTTPError {
	return httpError{httpStatus: httpStatus{code: statusCode, message: message}}
}

// Errorf returns a new HTTP error with a message constructed from a format string.
func Errorf(statusCode int, format string, args ...interface{}) HTTPError {
	return httpError{httpStatus: httpStatus{code: statusCode, message: fmt.Sprintf(format, args...)}}
}

// StatusFromError returns a HTTP status code for the given error.
func StatusFromError(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if he, ok := err.(HTTPStatus); ok {
		return he.HTTPStatusCode()
	}
	if spb, ok := status.FromError(err); ok {
		return runtime.HTTPStatusFromCode(spb.Code())
	}

	return http.StatusInternalServerError
}

// ErrorFromStatus returns a HTTP error for the given status, or nil if the status does not indicate an error.
func ErrorFromStatus(status HTTPStatus) HTTPError {
	if err, ok := status.(HTTPError); ok {
		return err
	}
	return nil
}

// WriteGRPCStyleError writes a gRPC-style error to an http response writer.
// It's useful when you have to write an http method.
func WriteGRPCStyleError(w http.ResponseWriter, c codes.Code, err error) {
	userErr := status.New(c, err.Error()).Proto()
	m := jsonpb.Marshaler{}

	w.WriteHeader(runtime.HTTPStatusFromCode(c))
	_ = m.Marshal(w, userErr)
}

// WriteGRPCStyleErrorf writes a gRPC-style error to an http response writer.
// It's useful when you have to write an http method.
func WriteGRPCStyleErrorf(w http.ResponseWriter, c codes.Code, format string, args ...interface{}) {
	WriteGRPCStyleError(w, c, fmt.Errorf(format, args...))
}

// WriteError writes the given error to the stream. If the error is a grpc status, the respective status proto will
// be generated with the adequate response code. Otherwise, if the error is an HTTPStatus, the HTTP status code will
// be used with a gRPC code of `Unknown`. If neither applies, a 200 OK with an empty message will be sent for `nil`,
// and 500 Internal Server Error with the appropriate message for other, non-nil errors.
func WriteError(w http.ResponseWriter, err error) {
	st, _ := status.FromError(err)
	w.WriteHeader(StatusFromError(err))
	_ = new(jsonpb.Marshaler).Marshal(w, st.Proto())
}

// WriteErrorf is a convenience method that is equivalent to calling
// `WriteError(w, Errorf(statusCode, format, args...)`.
func WriteErrorf(w http.ResponseWriter, statusCode int, format string, args ...interface{}) {
	WriteError(w, Errorf(statusCode, format, args...))
}
