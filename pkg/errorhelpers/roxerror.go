package errorhelpers

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RoxError is an error interface.
type RoxError interface {
	error
	Namespace() string
	GRPCCode() codes.Code
}

// ErrRox wraps a generic error and holds other detais.
// Implements RoxError interface.
type ErrRox struct {
	ns       string
	grpcCode codes.Code
	message  string
}

// NewRoxGRPCError returns RoxError interface pointing to the constructed ErrRox object with an associated GRPC error code.
// Does not record the stack. Wrap via errors.WithStack if needed.
func NewRoxGRPCError(ns string, grpcCode codes.Code, message string) RoxError {
	return ErrRox{
		ns:       ns,
		grpcCode: grpcCode,
		message:  message,
	}
}

// NewRoxError returns RoxError interface pointing to the constructed ErrRox object.
// Does not record the stack. Wrap via errors.WithStack if needed.
func NewRoxError(ns string, message string) RoxError {
	return ErrRox{
		ns:       ns,
		grpcCode: codes.Internal,
		message:  message,
	}
}

// Error returns the error message. Implements error interface.
func (e ErrRox) Error() string {
	return e.message
}

// GRPCCode returns the GRPC code, associtated with the error.
func (e ErrRox) GRPCCode() codes.Code {
	return e.grpcCode
}

// GRPCStatus can be called by grpc/status.FromError.
// Doesn't store the parent errors in the chain if the error is wrapped.
func (e ErrRox) GRPCStatus() *status.Status {
	return status.New(e.grpcCode, e.message)
}

// Namespace returns the error namespace. Allows for differentiating similar errors from different packages.
func (e ErrRox) Namespace() string {
	return e.ns
}

// IsRoxError unwraps provided error chain until finds a RoxError inside or reaches nil.
// If RoxError is found returns the interface pointer and true, or nil and false otherwise.
func IsRoxError(err error) (RoxError, bool) {
	if err == nil {
		return nil, false
	}
	if re, ok := err.(RoxError); ok {
		return re, true
	}
	return IsRoxError(errors.Unwrap(err))
}

// Is to be called by errors.Is. Returns true if the err chain has a rox error equal to this object.
// Note that errors.Is(nil, nil)==true whatever the interfaces.
func (e ErrRox) Is(err error) bool {
	re, ok := IsRoxError(err)
	return ok && re == e
}
