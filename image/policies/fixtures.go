package policies

import (
	"path/filepath"
	"runtime"

	"github.com/stackrox/rox/pkg/testutils"
)

// Directory returns the directory path at which these fixtures can be found in tests.
func Directory() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	ws := testutils.GetTestWorkspaceDir()

	// without bazel, we can use the absolute path directly.
	if dir[0] == '/' {
		return dir
	}

	// in bazel this path string is already qualified to the workspace, so we don't need to trim the prefix
	return filepath.Join(ws, dir)
}
