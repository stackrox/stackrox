package httputil

import (
	"net/http"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockResponseWriter struct {
	writtenStatusCodes int
}

func (rw *mockResponseWriter) Header() http.Header {
	return make(http.Header)
}

func (rw *mockResponseWriter) Write(data []byte) (int, error) {
	return len(data), nil
}

func (rw *mockResponseWriter) WriteHeader(statusCode int) {
	rw.writtenStatusCodes = statusCode
}

func TestWriteError(t *testing.T) {
	type testCase struct {
		name           string
		incomingErr    error
		expectedStatus int
	}
	tests := []testCase{
		{
			name:           "nil error maps to 200",
			incomingErr:    nil,
			expectedStatus: 200,
		},
		{
			name:           "HTTPError.code is propagated to response header",
			incomingErr:    NewError(404, "Origin: HTTPError"),
			expectedStatus: 404,
		},
		{
			name:           "gRPC's Status.code is propagated to response header",
			incomingErr:    status.New(codes.NotFound, "Origin: gRPC Status").Err(),
			expectedStatus: 404,
		},
		{
			name:           "Known internal error yields appropriate status in response header",
			incomingErr:    errors.Wrap(errox.NotFound, "Origin: known internal error"),
			expectedStatus: 404,
		},
		{
			name:           "Error of an unknown type yields 500",
			incomingErr:    errors.New("Origin: error of unknown type"),
			expectedStatus: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &mockResponseWriter{}
			WriteError(writer, tt.incomingErr)
			assert.Equal(t, tt.expectedStatus, writer.writtenStatusCodes)
		})
	}
}
