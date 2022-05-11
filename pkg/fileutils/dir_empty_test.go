package fileutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertPathReturns(t *testing.T, path string, retVal bool) {
	assert.Equal(t, retVal, DirExistsAndIsEmpty(path))
}

func TestDirEmpty(t *testing.T) {
	tempDir := t.TempDir()

	assertPathReturns(t, tempDir, true)
	randomFile := filepath.Join(tempDir, "RANDOM_FILE")
	assertPathReturns(t, randomFile, false)

	f, err := os.Create(randomFile)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	assertPathReturns(t, tempDir, false)
	assertPathReturns(t, randomFile, false)
}
