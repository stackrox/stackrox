package errorhelpers

import "github.com/pkg/errors"

var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")
)
