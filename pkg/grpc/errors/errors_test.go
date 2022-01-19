package errors

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type myError struct {
	code    codes.Code
	message string
}

func (e *myError) GRPCStatus() *status.Status {
	return status.New(e.code, e.message)
}

func (e *myError) Error() string {
	return e.message
}

func Test_unwrapGRPCStatus(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"NotFound", &myError{codes.NotFound, "not found"}, codes.NotFound},
		{"Wrapped", errors.Wrap(&myError{codes.NotFound, "not found"}, "wrapped"), codes.NotFound},
		{"Wrappped", errors.WithMessage(errors.Wrap(&myError{codes.NotFound, "not found"}, "wrapped"), "with message"), codes.NotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := unwrapGRPCStatus(tt.err)
			assert.Equal(t, s.Code(), tt.code)
		})
	}
}
