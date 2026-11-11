package complianceoperator

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	kubeAPIErr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func newTestRetrier() *retrier {
	return &retrier{
		maxRetries:     3,
		apiCallTimeout: 1 * time.Second,
		retryTimeout:   5 * time.Second,
	}
}

func TestCallWithRetry(t *testing.T) {
	tests := map[string]struct {
		fn      func(*atomic.Int32) retriableCall
		wantErr string
		calls   int32
	}{
		"succeeds on first attempt": {
			fn: func(counter *atomic.Int32) retriableCall {
				return func(_ context.Context) error {
					counter.Add(1)
					return nil
				}
			},
			calls: 1,
		},
		"retries on transient error then succeeds": {
			fn: func(counter *atomic.Int32) retriableCall {
				return func(_ context.Context) error {
					n := counter.Add(1)
					if n < 3 {
						return errors.New("transient")
					}
					return nil
				}
			},
			calls: 3,
		},
		"returns error after exhausting retries": {
			fn: func(counter *atomic.Int32) retriableCall {
				return func(_ context.Context) error {
					counter.Add(1)
					return errors.New("persistent failure")
				}
			},
			wantErr: "persistent failure",
			calls:   3,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := newTestRetrier()
			var counter atomic.Int32
			err := r.callWithRetry(context.Background(), tc.fn(&counter))
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.calls, counter.Load())
		})
	}
}

func TestCallWithRetryRespectsContextCancellation(t *testing.T) {
	r := newTestRetrier()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := r.callWithRetry(ctx, func(_ context.Context) error {
		return errors.New("should not retry after cancel")
	})
	require.Error(t, err)
}

func TestCallWithRetryWithOnConflictCallback(t *testing.T) {
	tests := map[string]struct {
		fnErr          error
		expectCallback bool
	}{
		"calls callback on K8s conflict error": {
			fnErr:          kubeAPIErr.NewConflict(schema.GroupResource{Group: "compliance.openshift.io", Resource: "scansettings"}, "test", errors.New("conflict")),
			expectCallback: true,
		},
		"calls callback on retryable error": {
			fnErr:          retry.MakeRetryable(errors.New("retryable")),
			expectCallback: true,
		},
		"does not call callback on non-conflict error": {
			fnErr:          errors.New("some other error"),
			expectCallback: false,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			r := newTestRetrier()
			r.maxRetries = 1

			var callbackCalled atomic.Bool
			err := r.callWithRetryWithOnConflictCallback(context.Background(),
				func(_ context.Context) error {
					return tc.fnErr
				},
				func(_ context.Context) error {
					callbackCalled.Store(true)
					return nil
				},
			)
			require.Error(t, err)
			assert.Equal(t, tc.expectCallback, callbackCalled.Load())
		})
	}
}

func TestOnConflictErrorWrapper(t *testing.T) {
	tests := map[string]struct {
		err     error
		wantNil bool
		nilFn   bool
		wantErr string
	}{
		"returns nil for non-conflict non-retryable error": {
			err:     errors.New("regular error"),
			wantNil: true,
		},
		"invokes callback on conflict error": {
			err:     kubeAPIErr.NewConflict(schema.GroupResource{}, "test", errors.New("conflict")),
			wantNil: true,
		},
		"returns error when callback is nil on conflict": {
			err:     kubeAPIErr.NewConflict(schema.GroupResource{}, "test", errors.New("conflict")),
			nilFn:   true,
			wantErr: "onConflictCall is 'nil'",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var fn onConflictCall
			if !tc.nilFn {
				fn = func(_ context.Context) error { return nil }
			}
			wrapper := onConflictErrorWrapper(fn)
			result := wrapper(context.Background(), tc.err)
			if tc.wantNil {
				assert.NoError(t, result)
			} else {
				require.Error(t, result)
				assert.Contains(t, result.Error(), tc.wantErr)
			}
		})
	}
}

func TestCallWithRetryNilFn(t *testing.T) {
	r := newTestRetrier()
	err := r.callWithRetry(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retriableCall is 'nil'")
}
