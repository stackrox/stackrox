package errors

import "github.com/stackrox/rox/pkg/errox"

var (
	// ErrAlreadyExists indicates that the object already exists.
	ErrAlreadyExists = errox.New(errox.CodeAlreadyExists, "central", "already exists")
)
