package errorhelpers

import (
	"fmt"
)

// ErrType is an enum of error types
type ErrType int

const (
	// ErrAlreadyExists is to be used when the object you are updating already exists
	ErrAlreadyExists ErrType = iota
)

// ErrorWrap allows us to attach extra metadata to error messages
type ErrorWrap struct {
	Type ErrType
	Msg  string
}

// Error returns the string form of the message
func (e *ErrorWrap) Error() string {
	return e.Msg
}

// Newf creates a new Error with the passed type and message
func Newf(t ErrType, template string, args ...interface{}) error {
	return &ErrorWrap{
		Type: t,
		Msg:  fmt.Sprintf(template, args...),
	}
}
