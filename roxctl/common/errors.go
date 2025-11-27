package common

import (
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/retry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	// ErrInvalidCommandOption indicates bad options provided by the user
	// during invocation of roxctl command.
	ErrInvalidCommandOption = errox.InvalidArgs.New("invalid command option")

	// ErrDeprecatedFlag is error factory for commands with deprecated flags.
	ErrDeprecatedFlag = func(oldFlag, newFlag string) errox.Error {
		return errox.InvalidArgs.Newf("specified deprecated flag %q and new flag %q at the same time", oldFlag, newFlag)
	}

	roxctlGrpcRetryPolicy = retry.AllGrpcCodesPolicy().WithNonRetryableCodes(
		codes.Unauthenticated,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.InvalidArgument,
	)
)

func MakeRetryable(err error) error {
	// Specific sentinel errors shouldn't be retried.
	if errox.IsAny(err, ErrInvalidCommandOption, errox.InvalidArgs, errox.NotAuthorized, errox.NoCredentials) {
		return err
	}

	// Check if this is a gRPC error
	_, ok := status.FromError(err)
	if !ok {
		// Not a gRPC error - retry all non-gRPC errors
		return retry.MakeRetryable(err)
	}

	// It's a gRPC error - check if it's retryable
	if roxctlGrpcRetryPolicy.ShouldRetry(err) {
		return retry.MakeRetryable(err)
	}

	// Non-retryable gRPC error
	return err
}
