package httputil

import (
	"net/http/httptest"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteError(t *testing.T) {
	type testCase struct {
		name                string
		incomingErr         error
		grpcCode            codes.Code
		expectedStatus      int
		expectedMessage     string
		expectedGRPCMessage string
	}
	tests := []testCase{
		{
			name:                "nil error maps to 200",
			incomingErr:         nil,
			grpcCode:            codes.OK,
			expectedStatus:      200,
			expectedMessage:     "{}",
			expectedGRPCMessage: "",
		},
		{
			name:                "HTTPError.code is propagated to response header",
			incomingErr:         NewError(404, "Origin: HTTPError"),
			grpcCode:            codes.NotFound,
			expectedStatus:      404,
			expectedMessage:     `{"code":13,"message":"Origin: HTTPError"}`,
			expectedGRPCMessage: `{"code":5,"message":"Origin: HTTPError"}`,
		},
		{
			name:                "gRPC's Status.code is propagated to response header",
			incomingErr:         status.New(codes.NotFound, "Origin: gRPC Status").Err(),
			grpcCode:            codes.NotFound,
			expectedStatus:      404,
			expectedMessage:     `{"code":5, "message":"Origin: gRPC Status"}`,
			expectedGRPCMessage: `{"code":5, "message":"rpc error: code = NotFound desc = Origin: gRPC Status"}`,
		},
		{
			name:                "Known internal error yields appropriate status in response header",
			incomingErr:         errors.Wrap(errox.NotFound, "Origin: known internal error"),
			grpcCode:            codes.NotFound,
			expectedStatus:      404,
			expectedMessage:     `{"code":5, "message":"Origin: known internal error: not found"}`,
			expectedGRPCMessage: `{"code":5, "message":"Origin: known internal error: not found"}`,
		},
		{
			name:                "Error of an unknown type yields 500",
			incomingErr:         errors.New("Origin: error of unknown type"),
			grpcCode:            codes.Internal,
			expectedStatus:      500,
			expectedMessage:     `{"code":13, "message":"Origin: error of unknown type"}`,
			expectedGRPCMessage: `{"code":13, "message":"Origin: error of unknown type"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := httptest.NewRecorder()
			WriteError(writer, tt.incomingErr)
			assert.Equal(t, tt.expectedStatus, writer.Code)
			data := writer.Body.String()
			if len(tt.expectedMessage) > 0 {
				assert.JSONEq(t, tt.expectedMessage, data)
			} else {
				assert.Equal(t, tt.expectedMessage, data)
			}

			// Note: writeGRPCStyleError panics on nil error
			if tt.incomingErr != nil {
				grpcWriter := httptest.NewRecorder()
				WriteGRPCStyleError(grpcWriter, tt.grpcCode, tt.incomingErr)
				assert.Equal(t, tt.expectedStatus, writer.Code)
				grpcData := grpcWriter.Body.String()
				if len(grpcData) > 0 {
					assert.JSONEq(t, tt.expectedGRPCMessage, grpcData)
				} else {
					assert.Equal(t, tt.expectedGRPCMessage, grpcData)
				}
			}
		})
	}
}
