package errorhelpers

import "github.com/pkg/errors"

var (
	// ErrAlreadyExists indicates that a object already exists.
	ErrAlreadyExists = errors.New("already exists")
)
