package invariant

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

func TestErrorToGrpcCodeInterceptor(t *testing.T) {
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
			assert.Equal(t, tt.err.Error(), err.Error())
		})
	}
}

func TestErrorToGrpcCodeStreamInterceptor(t *testing.T) {
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
			assert.Equal(t, tt.err.Error(), err.Error())
		})
	}
}
