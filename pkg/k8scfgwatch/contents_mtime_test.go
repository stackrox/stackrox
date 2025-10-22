package k8scfgwatch

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDirContentsMTime_EmptyDirectoryChanges tests that empty directory
// changes are detectable across multiple polls. This reproduces ROX-30992.
func TestDirContentsMTime_EmptyDirectoryChanges(t *testing.T) {
	dir := t.TempDir()

	// Initial state: empty directory
	mtime0, err := dirContentsMTime(dir)
	require.NoError(t, err)
	t.Logf("Empty dir mtime: %v", mtime0)

	// Sleep to ensure mtime difference
	time.Sleep(10 * time.Millisecond)

	// Add a file
	testFile := filepath.Join(dir, "test.yaml")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Poll 1: should detect file addition
	mtime1, err := dirContentsMTime(dir)
	require.NoError(t, err)
	t.Logf("After adding file: %v", mtime1)
	assert.NotEqual(t, mtime0, mtime1, "Adding file should change mtime")

	// Sleep to ensure mtime difference
	time.Sleep(10 * time.Millisecond)

	// Remove the file (simulating ConfigMap deletion)
	err = os.Remove(testFile)
	require.NoError(t, err)

	// Poll 2: should detect file removal
	mtime2, err := dirContentsMTime(dir)
	require.NoError(t, err)
	t.Logf("After removing file: %v", mtime2)
	assert.NotEqual(t, mtime1, mtime2, "Removing file should change mtime")

	// Poll 3: directory is still empty
	// BUG (before fix): returns zero, so mtime2 == mtime3 == zero (no change detected)
	// FIXED: returns dir.mtime, which differs from file.mtime but may still equal previous dir.mtime
	mtime3, err := dirContentsMTime(dir)
	require.NoError(t, err)
	t.Logf("Empty dir (second poll): %v", mtime3)

	// The critical assertion: empty directory should return a non-zero mtime
	assert.False(t, mtime3.IsZero(), "Empty directory should not return zero time")

	// For empty directory, mtime should be stable across polls (this is expected)
	assert.Equal(t, mtime2, mtime3, "Empty directory mtime should be stable")
}

// TestDirContentsMTime_FilesVsEmpty tests the transition from files to empty
func TestDirContentsMTime_FilesVsEmpty(t *testing.T) {
	dir := t.TempDir()

	// Create initial file
	testFile := filepath.Join(dir, "test.yaml")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Wait to ensure different mtimes
	time.Sleep(50 * time.Millisecond)

	// Get mtime with file present
	mtimeWithFile, err := dirContentsMTime(dir)
	require.NoError(t, err)
	require.False(t, mtimeWithFile.IsZero())

	// Wait to ensure different mtimes
	time.Sleep(50 * time.Millisecond)

	// Remove file
	err = os.Remove(testFile)
	require.NoError(t, err)

	// Get mtime after file removal
	mtimeEmpty, err := dirContentsMTime(dir)
	require.NoError(t, err)

	// KEY ASSERTION: Empty directory should return non-zero time
	assert.False(t, mtimeEmpty.IsZero(),
		"BUG: Empty directory returns zero time, preventing subsequent change detection")

	// The mtimes should be different (file removal changed something)
	assert.NotEqual(t, mtimeWithFile, mtimeEmpty,
		"Removing all files should result in different mtime")
}
