package sac

import (
	"errors"
)

var (
	// ErrPermissionDenied is the error when permission is denied for a SAC reason.
	ErrPermissionDenied = errors.New("permission denied")
)
