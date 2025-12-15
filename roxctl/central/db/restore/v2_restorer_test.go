package restore

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func TestExtractGRPCCodeFromHTTPError(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: codes.Unknown,
		},
		{
			name:         "error without JSON",
			err:          errors.New("some random error"),
			expectedCode: codes.Unknown,
		},
		{
			name:         "error without error message prefix",
			err:          errors.New(`{"code":13,"message":"test"}`),
			expectedCode: codes.Unknown,
		},
		{
			name:         "FailedPrecondition from HTTP 500",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":13,"message":"database restore failed: error processing file migration_version.yaml: rpc error: code = FailedPrecondition desc = Restoring from this version \"4.3.1\" is no longer supported, sequence number 194 matching software version 4.6"}`),
			expectedCode: codes.FailedPrecondition,
		},
		{
			name:         "PermissionDenied from HTTP 500",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":7,"message":"permission denied"}`),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "InvalidArgument from HTTP 500",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":3,"message":"invalid argument"}`),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "Internal error from HTTP 500 without nested error",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":13,"message":"internal error"}`),
			expectedCode: codes.Internal,
		},
		{
			name:         "Internal error with nested PermissionDenied",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":13,"message":"rpc error: code = PermissionDenied desc = access denied"}`),
			expectedCode: codes.PermissionDenied,
		},
		{
			name:         "Internal error with nested InvalidArgument",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {"code":13,"message":"rpc error: code = InvalidArgument desc = bad input"}`),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "malformed JSON",
			err:          errors.New(`received response code 500 Internal Server Error, but expected 2xx; error message: {not valid json`),
			expectedCode: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := extractGRPCCodeFromHTTPError(tt.err)
			assert.Equal(t, tt.expectedCode, code, "unexpected gRPC code extracted from error")
		})
	}
}

