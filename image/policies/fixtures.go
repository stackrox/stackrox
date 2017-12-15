package policies

import (
	"path/filepath"
	"runtime"
	"strings"

	"bitbucket.org/stack-rox/apollo/pkg/testutils"
)

// Directory returns the directory path at which these fixtures can be found in tests.
func Directory() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	ws := testutils.GetTestWorkspaceDir()
	// in bazel this path string is already qualified to the workspace, so we don't need to trim the prefix
	if dir[0] == '/' && strings.HasPrefix(dir, ws) {
		dir = dir[len(ws)+1:]
	}

	return filepath.Join(ws, dir)
}
