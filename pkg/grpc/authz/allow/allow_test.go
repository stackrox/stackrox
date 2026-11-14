package allow

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnonymous_Authorized(t *testing.T) {
	tests := map[string]struct {
		ctx        context.Context
		methodName string
	}{
		"nil context": {
			ctx:        nil,
			methodName: "",
		},
		"background context": {
			ctx:        context.Background(),
			methodName: "/some.Service/Method",
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
			assert.NoError(it, Anonymous().Authorized(tt.ctx, tt.methodName))
		})
	}
}

func TestAnonymous_ReturnsAuthorizer(t *testing.T) {
	a := Anonymous()
	assert.NotNil(t, a)
}
