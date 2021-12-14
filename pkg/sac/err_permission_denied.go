package sac

import (
	"github.com/stackrox/rox/pkg/errorhelpers"
)

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errorhelpers.New(errorhelpers.CodeResourceAccessDenied, "access to resource denied")
)
