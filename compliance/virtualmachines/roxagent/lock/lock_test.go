package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryLock_shouldAcquireLockOnValidTempPath(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	res, release, err := TryLock(lockPath)
	require.NoError(t, err)
	assert.Equal(t, Acquired, res)
	require.NotNil(t, release)

	release()

	res2, release2, err2 := TryLock(lockPath)
	require.NoError(t, err2)
	assert.Equal(t, Acquired, res2)
	require.NotNil(t, release2)
	release2()
}

func TestTryLock_shouldReportHeldWhenLockAlreadyAcquired(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	res1, release1, err1 := TryLock(lockPath)
	require.NoError(t, err1)
	require.Equal(t, Acquired, res1)
	require.NotNil(t, release1)
	defer release1()

	res2, release2, err2 := TryLock(lockPath)
	require.NoError(t, err2)
	assert.Equal(t, Held, res2)
	assert.Nil(t, release2)
}

func TestTryLock_shouldReleaseLockWhenReleaseFunctionCalled(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "test.lock")
	res, release, err := TryLock(lockPath)
	require.NoError(t, err)
	require.Equal(t, Acquired, res)
	require.NotNil(t, release)
	release()

	res2, release2, err2 := TryLock(lockPath)
	require.NoError(t, err2)
	assert.Equal(t, Acquired, res2)
	require.NotNil(t, release2)
	release2()
}

func TestTryLock_shouldCreateParentDirectoryIfMissing(t *testing.T) {
	tmp := t.TempDir()
	lockPath := filepath.Join(tmp, "subdir", "test.lock")
	parent := filepath.Join(tmp, "subdir")

	_, err := os.Stat(parent)
	require.True(t, os.IsNotExist(err), "parent must not exist before TryLock")

	res, release, err := TryLock(lockPath)
	require.NoError(t, err)
	assert.Equal(t, Acquired, res)
	require.NotNil(t, release)
	release()

	info, err := os.Stat(parent)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestTryLock_invalidInputs(t *testing.T) {
	tests := map[string]struct {
		lockPath  string
		setup     func(t *testing.T) string
		errSubstr string
	}{
		"should report unavailable for empty path": {
			lockPath:  "",
			errSubstr: "lock path is empty",
		},
		"should report unavailable for unwritable parent": {
			setup: func(t *testing.T) string {
				roDir := filepath.Join(t.TempDir(), "readonly")
				require.NoError(t, os.MkdirAll(roDir, 0o755))
				require.NoError(t, os.Chmod(roDir, 0o555))
				t.Cleanup(func() {
					_ = os.Chmod(roDir, 0o755)
				})
				return filepath.Join(roDir, "test.lock")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			path := tt.lockPath
			if tt.setup != nil {
				path = tt.setup(t)
			}
			res, release, err := TryLock(path)
			assert.Equal(t, Unavailable, res)
			assert.Nil(t, release)
			require.Error(t, err)
			if tt.errSubstr != "" {
				assert.ErrorContains(t, err, tt.errSubstr)
			}
		})
	}
}
