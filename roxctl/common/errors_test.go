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
