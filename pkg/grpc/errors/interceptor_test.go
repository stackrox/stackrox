package errors

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorToGrpcCodeInterceptor(t *testing.T) {
	tests := []struct {
		name    string
		handler grpc.UnaryHandler
		resp    interface{}
		err     error
	}{
		{
			name: "Error is nil -> do nothing, just pass through",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "OK", nil
			},
			resp: "OK", err: nil,
		},
		{
			name: "Error is already a gRPC status error (w/ status code) -> don't modify, just pass through",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "err", status.Error(codes.Canceled, "error message")
			},
			resp: "err", err: status.Error(codes.Canceled, "error message"),
		},
		{
			name: "Error is one of types from pkg/errorhelpers (ErrNotFound etc.) -> map to correct gRPC code, preserve error message",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "err", errors.Wrap(errorhelpers.ErrNotFound, "error message")
			},
			resp: "err", err: status.Error(codes.NotFound, errors.Wrap(errorhelpers.ErrNotFound, "error message").Error()),
		},
		{
			name: "Error is not a gRPC status error and not a known error type -> set error to internal",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "err", errors.New("some error")
			},
			resp: "err", err: status.Error(codes.Internal, "some error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := ErrorToGrpcCodeInterceptor(context.Background(), nil, nil, tt.handler)
			assert.Equal(t, tt.resp, resp)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}
			require.NotNil(t, err)
			assert.Equal(t, tt.err.Error(), err.Error())
		})
	}
}

func TestErrorToGrpcCodeStreamInterceptor(t *testing.T) {
	tests := []struct {
		name    string
		handler grpc.StreamHandler
		err     error
	}{
		{
			name: "Error is nil -> do nothing, just pass through",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			err: nil,
		},
		{
			name: "Error is already a gRPC status error (w/ status code) -> don't modify, just pass through",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return status.Error(codes.Canceled, "error message")
			},
			err: status.Error(codes.Canceled, "error message"),
		},
		{
			name: "Error is one of types from pkg/errorhelpers (ErrNotFound etc.) -> map to correct gRPC code, preserve error message",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return errors.Wrap(errorhelpers.ErrNotFound, "error message")
			},
			err: status.Error(codes.NotFound, errors.Wrap(errorhelpers.ErrNotFound, "error message").Error()),
		},
		{
			name: "Error is not a gRPC status error and not a known error type -> set error to internal",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return errors.New("some error")
			},
			err: status.Error(codes.Internal, "some error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrorToGrpcCodeStreamInterceptor(nil, nil, nil, tt.handler)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}
			require.NotNil(t, err)
			assert.Equal(t, tt.err.Error(), err.Error())
		})
	}
}

func TestPanicOnInvariantViolationUnaryInterceptor(t *testing.T) {
	tests := []struct {
		name    string
		handler grpc.UnaryHandler
		resp    interface{}
		err     error
		panics  bool
	}{
		{
			name: "Error is nil -> do nothing, just pass through",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "OK", nil
			},
			resp: "OK", err: nil,
			panics: false,
		},
		{
			name: "Error is ErrInvariantViolation -> panic",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "err", errorhelpers.ErrInvariantViolation
			},
			resp: nil, err: nil,
			panics: true,
		},
		{
			name: "Error is not ErrInvariantViolation -> do nothing, just pass through",
			handler: func(ctx context.Context, req interface{}) (interface{}, error) {
				return "err", errorhelpers.ErrNoCredentials
			},
			resp: "err", err: errorhelpers.ErrNoCredentials,
			panics: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					_, _ = PanicOnInvariantViolationUnaryInterceptor(context.Background(), nil, nil, tt.handler)
				}, "didn't panic")
				return
			}
			resp, err := PanicOnInvariantViolationUnaryInterceptor(context.Background(), nil, nil, tt.handler)
			assert.Equal(t, tt.resp, resp)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}
			require.NotNil(t, err)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}

func TestPanicOnInvariantViolationStreamInterceptor(t *testing.T) {
	tests := []struct {
		name    string
		handler grpc.StreamHandler
		err     error
		panics  bool
	}{
		{
			name: "Error is nil -> do nothing, just pass through",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			},
			err:    nil,
			panics: false,
		},
		{
			name: "Error is ErrInvariantViolation -> panic",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return errorhelpers.NewErrInvariantViolation("some explanation")
			},
			err:    nil,
			panics: true,
		},
		{
			name: "Error is not ErrInvariantViolation -> do nothing, just pass through",
			handler: func(srv interface{}, stream grpc.ServerStream) error {
				return errorhelpers.ErrNotFound
			},
			err:    errorhelpers.ErrNotFound,
			panics: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.panics {
				assert.Panics(t, func() {
					_ = PanicOnInvariantViolationStreamInterceptor(nil, nil, nil, tt.handler)
				}, "didn't panic")
				return
			}
			err := PanicOnInvariantViolationStreamInterceptor(nil, nil, nil, tt.handler)
			if tt.err == nil {
				assert.NoError(t, err)
				return
			}
			require.NotNil(t, err)
			assert.ErrorIs(t, err, tt.err)
		})
	}
}
