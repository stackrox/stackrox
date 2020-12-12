package templates

import (
	"fmt"
	"path/filepath"
	"runtime"
)

// Directory returns the directory path at which these fixtures can be found in tests.
func Directory() string {
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)

	if dir[0] != '/' {
		panic(fmt.Sprintf("directory should be an absolute path, is: %s", dir))
	}

	return dir
}
