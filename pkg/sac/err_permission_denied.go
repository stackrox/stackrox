package sac

import "github.com/stackrox/rox/pkg/errox"

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errox.New(errox.CodeResourceAccessDenied, "access to resource denied")
)
