package errorhelpers

import "github.com/pkg/errors"

var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrInvalidArgs indicates that a request has invalid arguments.
	ErrInvalidArgs = errors.New("invalid arguments")

	// ErrNotFound indicates that the requested object was not found.
	ErrNotFound = errors.New("not found")

	// ErrReferencedByAnotherObject indicates that the requested object cannot
	// be removed because it is referred to / in use by another object.
	ErrReferencedByAnotherObject = errors.New("referenced by another object")
)
