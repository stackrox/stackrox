package errox

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

// GetBaseSentinelError returns the lowest found sentinel error, or ServerError
// if none found.
func GetBaseSentinelError(err error) Error {
	var re Error
	for errors.As(err, &re) {
		err = errors.Unwrap(re)
	}
	if re, ok := (re).(*RoxError); ok {
		return re
	}
	return ServerError
}

func GetUserMessage(err error) string {
	var um *userMessage
	if errors.As(err, &um) {
		return um.UserMessage()
	}
	return ""
}
