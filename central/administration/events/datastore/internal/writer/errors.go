package writer

import (
	"github.com/pkg/errors"
)

// errWriteBufferExhausted indicates that the write buffer is out of capacity.
var errWriteBufferExhausted = errors.New("write buffer capacity exhausted")
