package common

import (
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/roxctl/common/flags"
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

// EnhanceConnectionError enhances connection errors with helpful guidance when
// the user has not explicitly configured an endpoint and connection fails.
func EnhanceConnectionError(err error) error {
	if err == nil {
		return nil
	}

	// Only enhance errors when using the default endpoint
	if flags.EndpointWasExplicitlyProvided() {
		return err
	}

	// Check if this is a connection-related error
	errMsg := err.Error()
	isConnectionError := strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "no such host") ||
		strings.Contains(errMsg, "dial tcp") ||
		strings.Contains(errMsg, "deadline exceeded")

	if !isConnectionError {
		return err
	}

	// Enhance the error with helpful guidance
	return errors.Wrapf(err,
		"Could not connect to Central at default endpoint (localhost:8443).\n"+
			"HINT: Configure the Central endpoint using the -e/--endpoint flag or ROX_ENDPOINT environment variable.\n"+
			"      Example: roxctl -e central.example.com:443 <command>")
}
