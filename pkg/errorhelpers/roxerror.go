package errorhelpers

// Code is the generic error code type.
type Code int

// Generic Rox error codes.
const (
	CodeOK Code = iota
	CodeAlreadyExists
	CodeInvalidArgs
	CodeNotFound
	CodeReferencedByAnotherObject
	CodeInvariantViolation
	CodeNoCredentials
	CodeNoValidRole
	CodeNotAuthorized
	CodeNoAuthzConfigured
	CodeResourceAccessDenied
	// When adding a new code, consider updating the existing translations to other codes, like GRPC.
)

// RoxError is the common error interface.
// Package errors are invited to implement this interface so the error translations
// could convert such errors to other types.
type RoxError interface {
	error
	Code() Code
}

type errRox struct {
	code    Code
	message string
}

// New returns a new RoxError.
func New(code Code, message string) RoxError {
	return &errRox{code, message}
}

// Error return error message. Implements error interface.
func (e *errRox) Error() string {
	return e.message
}

// Code returns error code. Implements RoxError interface.
func (e *errRox) Code() Code {
	return e.code
}
