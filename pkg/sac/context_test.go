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
		builtinScopedAuthz bool
	}{{
		ctx:                context.Background(),
		builtinScopedAuthz: false,
	}, {
		ctx:                SetContextPluginScopedAuthzEnabled(context.Background()),
		builtinScopedAuthz: true,
	}, {
		ctx:                context.WithValue(context.Background(), pluginScopedAuthzEnabled{}, struct{}{}),
		builtinScopedAuthz: true,
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.ctx), func(t *testing.T) {
			assert.Equal(t, tt.builtinScopedAuthz, IsContextPluginScopedAuthzEnabled(tt.ctx))
		})
	}
}
