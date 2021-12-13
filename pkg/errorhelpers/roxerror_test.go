package errorhelpers

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAsIs(t *testing.T) {
	testError := ErrAlreadyExists.Wrap(errors.New("test error"))
	tests := []struct {
		name string
		err  error
		re   RoxError
		ok   bool
	}{
		{"True rox errors", ErrNotFound, ErrNotFound, true},
		{"Nil error is not rox error", nil, nil, false},
		{"Custom standard error", errors.New("custom error"), nil, false},
		{"Wrapped known rox error", fmt.Errorf("wrapped rox error: %w", ErrAlreadyExists), ErrAlreadyExists, true},
		{"Custom rox error", testError, testError, true},
		{"Custom wrapped rox error", fmt.Errorf("wrapped custom: %w", testError), testError, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var re RoxError
			ok := errors.As(tt.err, &re)
			if ok != tt.ok {
				t.Errorf("IsRoxError() ok = %v, want %v", ok, tt.ok)
			}
			if errors.Is(re, tt.err) != ok && re != nil {
				t.Errorf("IsRoxError() re = %v (%T), want %v (%T)", re, re, tt.re, tt.re)
			}
		})
	}

	t.Run("Equals by value", func(t *testing.T) {
		if errors.Is(ErrNotFound, ErrNotAuthorized) {
			t.Errorf("Expected different code to be not equal")
		}
		if !errors.Is(ErrNotFound, ErrNotFound.Wraps("err conciled")) {
			t.Errorf("Expected errors with equal code to be equal")
		}
	})
}

func TestWrapsCode(t *testing.T) {
	e := ErrNotFound.Wraps("test message")
	var re RoxError
	if !errors.As(e, &re) {
		t.Errorf("Wraps() = %v, want %v", re, e)
	}
	if c := e.Code(); c != codes.NotFound {
		t.Errorf("Expected NotFound GRPC code, got: %d", c)
	}
}

func TestWrapf(t *testing.T) {
	e := ErrNotFound.Wrapf("test message")
	if e.Error() != "not found: test message" {
		t.Errorf("Expected \"not found: test message\", got %s", e.Error())
	}
	e1 := errors.Wrap(ErrNotFound, "test message")
	if e1.Error() != "test message: not found" {
		t.Errorf("Expected \"test message: not found\", got %s", e1.Error())
	}
}

func TestErrRox_GRPCStatus(t *testing.T) {
	s, ok := status.FromError(ErrAlreadyExists)
	if !ok {
		t.Error("Expected successful FromError")
	}
	if s.Code() != ErrAlreadyExists.Code() || s.Message() != ErrAlreadyExists.Error() {
		t.Errorf("Expected ErrAlreadyExists, got: %v", s)
	}
}
