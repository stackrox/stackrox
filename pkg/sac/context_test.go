package sac

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsContextBuiltinScopedAuthzEnabled(t *testing.T) {
	tests := []struct {
		ctx                context.Context
		sac                bool
		builtinScopedAuthz bool
	}{{
		ctx: context.Background(),
		sac: false, builtinScopedAuthz: false,
	}, {
		ctx: SetContextSACEnabled(context.Background()),
		sac: true, builtinScopedAuthz: false,
	}, {
		ctx: SetContextBuiltinScopedAuthzEnabled(context.Background()),
		sac: true, builtinScopedAuthz: true,
	}, {
		ctx: context.WithValue(context.Background(), builtinScopedAuthzEnabled{}, struct{}{}),
		sac: false, builtinScopedAuthz: false,
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.ctx), func(t *testing.T) {
			assert.Equal(t, tt.sac, IsContextSACEnabled(tt.ctx))
			assert.Equal(t, tt.builtinScopedAuthz, IsContextBuiltinScopedAuthzEnabled(tt.ctx))
		})
	}
}
