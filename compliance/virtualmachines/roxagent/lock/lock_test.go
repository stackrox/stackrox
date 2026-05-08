package lock

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryLock_shouldAcquireAndReacquireAfterRelease(t *testing.T) {
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
		errSubstr string
	}{
		"should report unavailable for empty path": {
			lockPath:  "",
			errSubstr: "lock path is empty",
		},
		"should report unavailable when parent is not a directory": {
			// /dev/null is a character device, not a directory. MkdirAll fails
			// with ENOTDIR even when running as root, unlike chmod-based tests
			// where root bypasses permission checks.
			lockPath:  "/dev/null/test.lock",
			errSubstr: "not a directory",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			res, release, err := TryLock(tt.lockPath)
			assert.Equal(t, Unavailable, res)
			assert.Nil(t, release)
			require.Error(t, err)
			assert.ErrorContains(t, err, tt.errSubstr)
		})
	}
}
