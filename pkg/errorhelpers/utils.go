package errorhelpers

import "github.com/pkg/errors"

// IsAny returns a bool if it matches any of the target errors
// This helps consolidate code from
// errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.ErrClosedPipe)
// to errors.IsAny(err, io.EOF. io.ErrUnexpectedEOF, io.ErrClosedPipe)
func IsAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}
