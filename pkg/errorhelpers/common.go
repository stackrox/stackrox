package errorhelpers

import (
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// NOTE: These errors can (and should?) be moved to appropriate packages. If an
// error becomes common across multiple packages and/or components, it should be
// moved to the _true_ sentinels list, currently in "custom_types.go".
var (
	// ErrNoAuthzConfigured occurs if authorization is not implemented for a
	// service. This is a programming error.
	ErrNoAuthzConfigured = OverrideMessage(ErrInvariantViolation, "service authorization is misconfigured")

	// ErrNoCredentials occurs if no credentials can be found.
	ErrNoCredentials = OverrideMessage(ErrNotAuthenticated, "credentials not found")

	// ErrNoValidRole occurs if no valid role can be found for user.
	ErrNoValidRole = OverrideMessage(ErrNotAuthenticated, "access for this user is not authorized: no valid role; please contact a system administrator")
)

func Explain(sentinel error, explanation string) error {
	return fmt.Errorf("%w: %s", sentinel, explanation)
}

func OverrideMessage(sentinel error, message string) error {
	if sentinel == nil {
		return nil
	}
	overridden := &overrideMessage{
		cause: sentinel,
		msg:   message,
	}
	return errors.WithStack(overridden)
}

func OverrideMessagef(sentinel error, format string, args ...interface{}) error {
	if sentinel == nil {
		return nil
	}
	overridden := &overrideMessage{
		cause: sentinel,
		msg:   fmt.Sprintf(format, args...),
	}
	return errors.WithStack(overridden)
}

// This differs from `pkg/errors:withMessage{}` in that the underlying error's
// message is dropped.
type overrideMessage struct {
	cause error
	msg   string
}
func (om *overrideMessage) Error() string { return om.msg }
func (om *overrideMessage) Unwrap() error { return om.cause }
func (om *overrideMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%+v\n", om.Unwrap())
			io.WriteString(s, om.Error())
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, om.Error())
	}
}

////////////////////////////////////////////////////////////////////////////////
// Consider removing the functions below in favour of direct use of           //
// `Explain()`.                                                               //
//

// NewErrNotAuthorized wraps ErrNotAuthorized into an explanation.
func NewErrNotAuthorized(explanation string) error {
	return Explain(ErrNotAuthorized, explanation)
}

// NewErrNoCredentials wraps ErrNoCredentials into an explanation.
func NewErrNoCredentials(explanation string) error {
	return Explain(ErrNoCredentials, explanation)
}

// NewErrInvariantViolation wraps ErrInvariantViolation into an explanation.
func NewErrInvariantViolation(explanation string) error {
	return Explain(ErrInvariantViolation, explanation)
}

// NewErrInvalidArgs wraps ErrInvalidArgs into an explanation.
func NewErrInvalidArgs(explanation string) error {
	return Explain(ErrInvalidArgs, explanation)
}
