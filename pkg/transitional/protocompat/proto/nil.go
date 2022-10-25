package proto

import "errors"

var (
	// ErrNil indicates that a function was attempted to be called on a nil protobuf message.
	ErrNil = errors.New("proto message is nil")
)
