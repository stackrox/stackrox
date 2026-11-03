package deny

import (
	"context"
	"testing"

	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFunc(t *testing.T) {
	tests := map[string]struct {
		ctx context.Context
	}{
		"background context is passed through": {
			ctx: context.Background(),
		},
		"context with value is passed through unchanged": {
			ctx: context.WithValue(context.Background(), struct{ key string }{"k"}, "v"),
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			got, err := AuthFunc(tt.ctx)
			assert.Equal(it, tt.ctx, got)
			assert.ErrorIs(it, err, errox.NoAuthzConfigured)
		})
	}
}

func TestEveryone_Authorized(t *testing.T) {
	tests := map[string]struct {
		ctx        context.Context
		methodName string
	}{
		"background context": {
			ctx:        context.Background(),
			methodName: "/some.Service/Method",
		},
		"nil context": {
			ctx:        nil,
			methodName: "",
		},
		"context with value": {
			ctx:        context.WithValue(context.Background(), struct{ key string }{"k"}, "v"),
			methodName: "/some.Service/Method",
		},
		"empty method name": {
			ctx:        context.Background(),
			methodName: "",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(it *testing.T) {
			assert.ErrorIs(it, Everyone().Authorized(tt.ctx, tt.methodName), errox.NoAuthzConfigured)
		})
	}
}

func TestEveryone_ReturnsAuthorizer(t *testing.T) {
	a := Everyone()
	require.NotNil(t, a)
}
