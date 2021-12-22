package errox

import (
	"strings"

	"github.com/pkg/errors"
)

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
	Namespace() string
	Code() Code
}

type errRox struct {
	code      Code
	namespace string
	message   string
}

// New returns a new RoxError.
func New(code Code, namespace, message string) RoxError {
	return &errRox{code, namespace, message}
}

// Error return error message. Implements error interface.
func (e *errRox) Error() string {
	return e.message
}

// Code returns error code. Implements RoxError interface.
func (e *errRox) Code() Code {
	return e.code
}

// Namespace returns error namespace. Implements RoxError interface.
func (e *errRox) Namespace() string {
	return e.namespace
}

// Is returns true if err wraps the same error code and comes from the same or higher namespace
func (e *errRox) Is(err error) bool {
	var re RoxError
	if errors.As(err, &re) {
		return e.code == re.Code() && strings.HasPrefix(e.namespace, re.Namespace())
	}
	return false
}
