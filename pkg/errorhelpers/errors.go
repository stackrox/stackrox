package errorhelpers

import (
	"github.com/stackrox/rox/pkg/errox"
)

// Deprecated: use the errox errors and functions instead of these aliases.
var (
	ErrNotFound = errox.NotFound

	GenericNoValidRole       = errox.GenericNoValidRole
	NewErrNotAuthorized      = errox.NewErrNotAuthorized
	NewErrNoCredentials      = errox.NewErrNoCredentials
	NewErrInvariantViolation = errox.NewErrInvariantViolation
	NewErrInvalidArgs        = errox.NewErrInvalidArgs
)
