package index

import (
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/lock"
	"github.com/stretchr/testify/assert"
)

func TestRunWithLock_orchestration(t *testing.T) {
	tests := map[string]struct {
		lockResult lock.Result
		wantScan   bool
		wantErrMsg string
	}{
		"should run scan when lock acquired": {
			lockResult: lock.Acquired,
			wantScan:   true,
		},
		"should skip scan when lock already held": {
			lockResult: lock.Held,
			wantScan:   false,
		},
		"should run scan in degraded mode when lock unavailable": {
			lockResult: lock.Unavailable,
			wantScan:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var scanCalled bool
			scanFn := func() error {
				scanCalled = true
				return nil
			}
			onHeld := func() error { return nil }
			onUnavailable := func(_ error) error { return scanFn() }

			lockDir := t.TempDir()
			lockPath := lockDir + "/test.lock"

			switch tt.lockResult {
			case lock.Acquired:
				err := lock.RunWithLock(lockPath, scanFn, onHeld, onUnavailable)
				assert.NoError(t, err)
			case lock.Held:
				res, release, err := lock.TryLock(lockPath)
				assert.Equal(t, lock.Acquired, res)
				assert.NoError(t, err)
				defer release()
				err = lock.RunWithLock(lockPath, scanFn, onHeld, onUnavailable)
				assert.NoError(t, err)
			case lock.Unavailable:
				err := lock.RunWithLock("/proc/not-writable/test.lock", scanFn, onHeld, onUnavailable)
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.wantScan, scanCalled)
		})
	}
}
