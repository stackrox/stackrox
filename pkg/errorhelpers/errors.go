package errorhelpers

import (
	"github.com/stackrox/rox/pkg/errox"
)

// TODO: make use of the errox errors and functions instead of these aliases.
var (
	ErrAlreadyExists             = errox.AlreadyExists
	ErrInvalidArgs               = errox.InvalidArgs
	ErrNotFound                  = errox.NotFound
	ErrReferencedByAnotherObject = errox.ReferencedByAnotherObject
	ErrInvariantViolation        = errox.InvariantViolation
	ErrNoCredentials             = errox.NoCredentials
	ErrNoValidRole               = errox.NoValidRole
	ErrNotAuthorized             = errox.NotAuthorized
	ErrNoAuthzConfigured         = errox.NoAuthzConfigured

	GenericNoValidRole       = errox.GenericNoValidRole
	NewErrNotAuthorized      = errox.NewErrNotAuthorized
	NewErrNoCredentials      = errox.NewErrNoCredentials
	NewErrInvariantViolation = errox.NewErrInvariantViolation
	NewErrInvalidArgs        = errox.NewErrInvalidArgs
)
