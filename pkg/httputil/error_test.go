package httputil

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil/mock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
			writer := mock.NewResponseWriter()
			WriteError(writer, tt.incomingErr)
			assert.Equal(t, tt.expectedStatus, writer.Code)
		})
	}
}
