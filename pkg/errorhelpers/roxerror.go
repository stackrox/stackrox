package errorhelpers

import (
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RoxError is an error interface.
type RoxError interface {
	error
	GRPCCode() codes.Code
}

// Implements RoxError interface.
type errRox struct {
	grpcCode codes.Code
	message  string
}

// NewWithCode returns a RoxError with the supplied GRPC error code and message.
func NewWithCode(grpcCode codes.Code, message string) RoxError {
	return &errRox{
		grpcCode: grpcCode,
		message:  message,
	}
}

// New returns a RoxError with the Internal GRPC code and supplied message.
func New(message string) RoxError {
	return NewWithCode(codes.Internal, message)
}

// Error returns the error message. Implements error interface.
func (e *errRox) Error() string {
	return e.message
}

// GRPCCode returns the GRPC code, associtated with the error.
func (e *errRox) GRPCCode() codes.Code {
	return e.grpcCode
}

// GRPCStatus can be called by grpc/status.FromError.
// Doesn't store the parent errors in the chain if the error is wrapped.
func (e *errRox) GRPCStatus() *status.Status {
	return status.New(e.grpcCode, e.message)
}

// Is called by errors.Is(err, target).
// Returns true if the target error chain has a RoxError with the same code and message.
func (e *errRox) Is(target error) bool {
	var re RoxError
	return errors.As(target, &re) && e.grpcCode == re.GRPCCode() && e.message == re.Error()
}
