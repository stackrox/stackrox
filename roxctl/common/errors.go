package common

import (
	"slices"

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

	nonRetryCodes = []codes.Code{
		codes.Unauthenticated,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.InvalidArgument,
	}
)

// MakeRetryable makes an error retryable based on the error type.
func MakeRetryable(err error) error {
	// Specific sentinel errors shouldn't be retried.
	if errox.IsAny(err, ErrInvalidCommandOption, errox.InvalidArgs, errox.NotAuthorized, errox.NoCredentials) {
		return err
	}

	s, ok := status.FromError(err)
	// Retry all errors that cannot be mapped to a GRPC code and aren't already explicitly skipped already.
	if !ok {
		return retry.MakeRetryable(err)
	}

	// Do not retry errors with specific GRPC codes such as unauthenticated etc.
	if slices.Contains(nonRetryCodes, s.Code()) {
		return err
	}

	// Mark all other errors as retryable.
	return retry.MakeRetryable(err)
}
