package sac

import (
	"github.com/stackrox/rox/pkg/errox"
)

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errox.NotAuthorized.New("access to resource denied")
)
