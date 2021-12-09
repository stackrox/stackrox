package errorhelpers

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsRoxError(t *testing.T) {
	testError := NewRoxError("test", "test error")
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
			re, ok := IsRoxError(tt.err)
			if ok != tt.ok {
				t.Errorf("IsRoxError() ok = %v, want %v", ok, tt.ok)
			}
			if errors.Is(re, tt.err) != ok && re != nil {
				t.Errorf("IsRoxError() re = %v, want %v", re, tt.re)
			}
		})
	}

	t.Run("As wrapped ok", func(t *testing.T) {
		target := ErrRox{}
		if !errors.As(fmt.Errorf("wrapped %w", testError), &target) {
			t.Errorf("Expected to get wrapped error as testError")
		} else if target.Error() != "test error" {
			t.Errorf("Expected to get %v, got %v", testError, target)
		}
	})
	t.Run("As wrapped ko", func(t *testing.T) {
		target := ErrRox{}
		if errors.As(fmt.Errorf("wrapper %w", errors.New("some error")), &target) {
			t.Errorf("Expected to not get wrapped error as testError")
		}
	})
}

func TestNewRoxError(t *testing.T) {
	e := NewRoxError("test", "test message")
	if re, ok := IsRoxError(e); !ok {
		t.Errorf("NewRoxError() = %v, want %v", re, e)
	}
	if c := e.GRPCCode(); c != codes.Internal {
		t.Errorf("Expected Internal GRPC code, got: %d", c)
	}
	if ns := e.Namespace(); ns != "test" {
		t.Errorf("Expected test namespace, got: %s", ns)
	}
}

func TestNewGRPCRoxError(t *testing.T) {
	e := NewRoxError("test", "test message")
	if re, ok := IsRoxError(e); !ok {
		t.Errorf("NewRoxError() = %v, want %v", re, e)
	}
	if c := e.GRPCCode(); c != codes.Internal {
		t.Errorf("Expected Internal GRPC code, got: %d", c)
	}
	if ns := e.Namespace(); ns != "test" {
		t.Errorf("Expected test namespace, got: %s", ns)
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
