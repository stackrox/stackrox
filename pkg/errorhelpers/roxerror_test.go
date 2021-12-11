package errorhelpers

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAsIs(t *testing.T) {
	testError := New("test error")
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
		e1 := NewWithGRPCCode(codes.Canceled, "err cancelled")
		e2 := NewWithGRPCCode(codes.Canceled, "err cancelled")
		e3 := NewWithGRPCCode(codes.Canceled, "err conciled")
		if !errors.Is(e1, e2) {
			t.Errorf("Expected equal errors to be equal")
		}
		if errors.Is(e1, e3) {
			t.Errorf("Expected not equal errors to be not equal")
		}
	})
}

func TestNewWithGRPCCode(t *testing.T) {
	e := New("test message")
	var re RoxError
	if !errors.As(e, &re) {
		t.Errorf("New() = %v, want %v", re, e)
	}
	if c := e.GRPCCode(); c != codes.Internal {
		t.Errorf("Expected Internal GRPC code, got: %d", c)
	}
}

func TestNew(t *testing.T) {
	e := New("test message")
	var re RoxError
	if !errors.As(e, &re) {
		t.Errorf("New() = %v, want %v", re, e)
	}
	if c := e.GRPCCode(); c != codes.Internal {
		t.Errorf("Expected Internal GRPC code, got: %d", c)
	}
}

func TestErrRox_GRPCStatus(t *testing.T) {
	s, ok := status.FromError(ErrAlreadyExists)
	if !ok {
		t.Error("Expected successful FromError")
	}
	if s.Code() != ErrAlreadyExists.GRPCCode() || s.Message() != ErrAlreadyExists.Error() {
		t.Errorf("Expected ErrAlreadyExists, got: %v", s)
	}
}
