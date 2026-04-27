package index

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runWithLock mirrors runSingleWithLock's lock switch logic but invokes scanFn instead of RunSingle.
func runWithLock(lockPath string, scanFn func() error) (lock.Result, error) {
	res, release, lockErr := lock.TryLock(lockPath)
	switch res {
	case lock.Acquired:
		defer release()
		return res, scanFn()
	case lock.Held:
		return res, nil
	case lock.Unavailable:
		_ = lockErr
		return res, scanFn()
	default:
		return res, fmt.Errorf("unexpected lock result: %d", res)
	}
}

func TestRunWithLock_orchestration(t *testing.T) {
	tests := map[string]struct {
		setup      func(t *testing.T) (lockPath string, cleanup func())
		assertions func(t *testing.T, scanCalled bool, res lock.Result, err error)
	}{
		"should run scan when lock acquired": {
			setup: func(t *testing.T) (string, func()) {
				return filepath.Join(t.TempDir(), "test.lock"), nil
			},
			assertions: func(t *testing.T, scanCalled bool, res lock.Result, err error) {
				assert.True(t, scanCalled)
				assert.Equal(t, lock.Acquired, res)
				assert.NoError(t, err)
			},
		},
		"should skip scan when lock already held": {
			setup: func(t *testing.T) (string, func()) {
				lockPath := filepath.Join(t.TempDir(), "test.lock")
				res, release, err := lock.TryLock(lockPath)
				require.NoError(t, err)
				require.Equal(t, lock.Acquired, res)
				require.NotNil(t, release)
				return lockPath, release
			},
			assertions: func(t *testing.T, scanCalled bool, res lock.Result, err error) {
				assert.False(t, scanCalled)
				assert.Equal(t, lock.Held, res)
				assert.NoError(t, err)
			},
		},
		"should run scan in degraded mode when lock unavailable": {
			setup: func(t *testing.T) (string, func()) {
				roDir := filepath.Join(t.TempDir(), "readonly")
				require.NoError(t, os.MkdirAll(roDir, 0o755))
				require.NoError(t, os.Chmod(roDir, 0o555))
				cleanup := func() {
					_ = os.Chmod(roDir, 0o755)
				}
				return filepath.Join(roDir, "test.lock"), cleanup
			},
			assertions: func(t *testing.T, scanCalled bool, res lock.Result, err error) {
				assert.True(t, scanCalled)
				assert.Equal(t, lock.Unavailable, res)
				assert.NoError(t, err)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			lockPath, cleanup := tt.setup(t)
			if cleanup != nil {
				defer cleanup()
			}

			var scanCalled bool
			res, err := runWithLock(lockPath, func() error {
				scanCalled = true
				return nil
			})
			tt.assertions(t, scanCalled, res, err)
		})
	}
}
