package kocache

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_offlineCtrl(t *testing.T) {
	tests := map[string]struct {
		parentCtx          context.Context
		startOnline        bool
		transition         func(o *offlineCtrl)
		wantOnline         bool
		wantCtxCancelCause error
	}{
		"Starting with parent ctx as nil should take background ctx as parent": {
			parentCtx:          nil,
			startOnline:        true,
			transition:         func(o *offlineCtrl) {},
			wantOnline:         true,
			wantCtxCancelCause: nil,
		},
		"Starting online should report being online": {
			parentCtx:          context.Background(),
			startOnline:        true,
			transition:         func(o *offlineCtrl) {},
			wantOnline:         true,
			wantCtxCancelCause: nil,
		},
		"Starting offline should report being offline": {
			parentCtx:          context.Background(),
			startOnline:        false,
			transition:         func(o *offlineCtrl) {},
			wantOnline:         false,
			wantCtxCancelCause: errCentralUnreachable,
		},
		"Starting online and transitioning to online should report being online": {
			parentCtx:   context.Background(),
			startOnline: true,
			transition: func(o *offlineCtrl) {
				o.GoOnline()
			},
			wantOnline:         true,
			wantCtxCancelCause: nil,
		},
		"Starting online and transitioning to offline should report being offline": {
			parentCtx:   context.Background(),
			startOnline: true,
			transition: func(o *offlineCtrl) {
				o.GoOffline()
			},
			wantOnline:         false,
			wantCtxCancelCause: errCentralUnreachable,
		},
		"Starting offline and transitioning to online should report being online": {
			parentCtx:   context.Background(),
			startOnline: false,
			transition: func(o *offlineCtrl) {
				o.GoOnline()
			},
			wantOnline:         true,
			wantCtxCancelCause: nil,
		},
		"Starting offline and transitioning to offline should report being offline": {
			parentCtx:   context.Background(),
			startOnline: false,
			transition: func(o *offlineCtrl) {
				o.GoOffline()
			},
			wantOnline:         false,
			wantCtxCancelCause: errCentralUnreachable,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			o := newOfflineCtrl(tt.parentCtx, tt.startOnline)
			if tt.transition != nil {
				tt.transition(o)
			}
			switch tt.wantOnline {
			case true:
				assert.True(t, o.IsOnline())
				assert.NoError(t, o.Context().Err())
			case false:
				assert.False(t, o.IsOnline())
				assert.Error(t, o.Context().Err())
				assert.ErrorIs(t, o.Context().Err(), context.Canceled)
				assert.ErrorIs(t, context.Cause(o.Context()), tt.wantCtxCancelCause)
			}
		})
	}
}
