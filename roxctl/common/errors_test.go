package common

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMakeRetryable(t *testing.T) {
	cases := []struct {
		err       error
		retryable bool
	}{
		{
			err: errox.InvalidArgs,
		},
		{
			err: errox.NotAuthorized,
		},
		{
			err: errox.NoCredentials,
		},
		{
			err: ErrInvalidCommandOption,
		},
		{
			err:       errox.ReferencedByAnotherObject,
			retryable: true,
		},
		{
			err:       errors.New("some error"),
			retryable: true,
		},
		{
			err: status.Error(codes.Unauthenticated, "some error"),
		},
		{
			err: status.Error(codes.AlreadyExists, "some error"),
		},

		{
			err: status.Error(codes.PermissionDenied, "some error"),
		},
		{
			err: status.Error(codes.InvalidArgument, "some error"),
		},
		{
			err:       status.Error(codes.DeadlineExceeded, "some error"),
			retryable: true,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("case-%d", i), func(t *testing.T) {
			err := MakeRetryable(c.err)
			assert.Equal(t, c.retryable, retry.IsRetryable(err))
		})
	}
}

func TestEnhanceConnectionError(t *testing.T) {
	cases := []struct {
		name              string
		err               error
		setEndpointEnv    bool
		expectEnhancement bool
	}{
		{
			name:              "nil error returns nil",
			err:               nil,
			setEndpointEnv:    false,
			expectEnhancement: false,
		},
		{
			name:              "connection refused with default endpoint",
			err:               errors.New("dial tcp [::1]:8443: connect: connection refused"),
			setEndpointEnv:    false,
			expectEnhancement: true,
		},
		{
			name:              "connection refused with explicit endpoint",
			err:               errors.New("dial tcp [::1]:8443: connect: connection refused"),
			setEndpointEnv:    true,
			expectEnhancement: false,
		},
		{
			name:              "i/o timeout with default endpoint",
			err:               errors.New("dial tcp 192.168.1.1:443: i/o timeout"),
			setEndpointEnv:    false,
			expectEnhancement: true,
		},
		{
			name:              "no such host with default endpoint",
			err:               errors.New("dial tcp: lookup invalid.local: no such host"),
			setEndpointEnv:    false,
			expectEnhancement: true,
		},
		{
			name:              "deadline exceeded with default endpoint",
			err:               errors.New("context deadline exceeded"),
			setEndpointEnv:    false,
			expectEnhancement: true,
		},
		{
			name:              "wrapped connection error with default endpoint",
			err:               errors.Wrap(errors.New("dial tcp [::1]:8443: connect: connection refused"), "error when doing http request"),
			setEndpointEnv:    false,
			expectEnhancement: true,
		},
		{
			name:              "authentication error with default endpoint",
			err:               errors.New("authentication failed: invalid credentials"),
			setEndpointEnv:    false,
			expectEnhancement: false,
		},
		{
			name:              "TLS error with default endpoint",
			err:               errors.New("x509: certificate signed by unknown authority"),
			setEndpointEnv:    false,
			expectEnhancement: false,
		},
		{
			name:              "generic error with default endpoint",
			err:               errors.New("some other error"),
			setEndpointEnv:    false,
			expectEnhancement: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment to simulate explicit or default endpoint
			if tc.setEndpointEnv {
				t.Setenv("ROX_ENDPOINT", "central.example.com:443")
			} else {
				t.Setenv("ROX_ENDPOINT", "")
			}

			originalErr := tc.err
			result := EnhanceConnectionError(tc.err)

			if tc.err == nil {
				assert.Nil(t, result)
				return
			}

			if tc.expectEnhancement {
				// Should contain the hint message
				assert.Contains(t, result.Error(), "Could not connect to Central at default endpoint")
				assert.Contains(t, result.Error(), "HINT: Configure the Central endpoint")
				assert.Contains(t, result.Error(), "Example: roxctl -e central.example.com:443")
				// Original error should still be present
				assert.Contains(t, result.Error(), originalErr.Error())
			} else {
				// Should be unchanged
				assert.Equal(t, originalErr, result)
			}
		})
	}
}
