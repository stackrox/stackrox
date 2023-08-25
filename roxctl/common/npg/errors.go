package npg

import (
	"errors"
)

var (
	// ErrErrors errors indicator message
	ErrErrors = errors.New("there were errors during execution")
	// ErrWarnings warnings indicator message
	ErrWarnings = errors.New("there were warnings during execution")
)
