package sac

import (
	"errors"
)

var (
	// ErrResourceAccessDenied is the error when permission is denied for a SAC reason.
	ErrResourceAccessDenied = errors.New("access to resource denied")
)
