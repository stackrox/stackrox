package index

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/compliance/virtualmachines/roxagent/lock"
	"github.com/stretchr/testify/assert"
)

// stubTryLock returns a tryLockFn that always produces the given lock outcome.
// This decouples orchestration tests from the filesystem so they work
// identically regardless of UID (including root in CI containers).
func stubTryLock(res lock.Result, err error) func() (lock.Result, func(), error) {
	return func() (lock.Result, func(), error) {
		var release func()
		if res == lock.Acquired {
			release = func() {}
		}
		return res, release, err
	}
}

func TestRunWithLock_orchestration(t *testing.T) {
	tests := map[string]struct {
		tryLockFn  func() (lock.Result, func(), error)
		wantScan   bool
		wantErrMsg string
	}{
		"should run scan when lock acquired": {
			tryLockFn: stubTryLock(lock.Acquired, nil),
			wantScan:  true,
		},
		"should skip scan when lock already held": {
			tryLockFn: stubTryLock(lock.Held, nil),
			wantScan:  false,
		},
		"should run scan in degraded mode when lock unavailable": {
			tryLockFn: stubTryLock(lock.Unavailable, errors.New("permission denied")),
			wantScan:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var scanCalled bool
			err := runWithLock("unused", func() error {
				scanCalled = true
				return nil
			}, tt.tryLockFn)

			assert.Equal(t, tt.wantScan, scanCalled)
			if tt.wantErrMsg != "" {
				assert.ErrorContains(t, err, tt.wantErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
