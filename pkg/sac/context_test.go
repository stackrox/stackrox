package sac

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsContextSACV2Enabled(t *testing.T) {

	tests := []struct {
		ctx   context.Context
		sac   bool
		sacV2 bool
	}{{
		ctx: context.Background(),
		sac: false, sacV2: false,
	}, {
		ctx: SetContextSACEnabled(context.Background()),
		sac: true, sacV2: false,
	}, {
		ctx: SetContextSACV2Enabled(context.Background()),
		sac: true, sacV2: true,
	}, {
		ctx: context.WithValue(context.Background(), sacV2Enabled{}, struct{}{}),
		sac: false, sacV2: false,
	},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.ctx), func(t *testing.T) {
			assert.Equal(t, tt.sac, IsContextSACEnabled(tt.ctx))
			assert.Equal(t, tt.sacV2, IsContextSACV2Enabled(tt.ctx))
		})
	}

}
