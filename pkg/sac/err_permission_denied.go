package sac

import (
	"github.com/stackrox/rox/pkg/errorhelpers"
)

// TODO(alexr): Rename this file to "errors.go"

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errorhelpers.OverrideMessage(errorhelpers.ErrNotAuthorized, "access to resource denied")
)
