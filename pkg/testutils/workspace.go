package testutils

import (
	"path/filepath"
	"runtime"

	"github.com/stretchr/testify/require"
)

// GetTestWorkspaceDir returns the root directory of our workspace we're building, which may or may not be under GOPATH
func GetTestWorkspaceDir(t T) string {
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok, "could not get source file path")
	return filepath.Join(filepath.Dir(filename), "../..")
}
