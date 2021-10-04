package fileutils

import (
	"io"
	"os"

	"github.com/stackrox/rox/pkg/utils"
)

// DirExistsAndIsEmpty returns whether the given path is an existing directory, and exists.
// It is more efficient than calling os.ReadDir for large directories.
// It returns false unless it determines for certain that the directory exists and is empty --
// in particular, this means that any underlying filesystem-related errors are swallowed.
// Callers needing more fine-grained information will not be able to use this function.
func DirExistsAndIsEmpty(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer utils.IgnoreError(f.Close)
	_, err = f.Readdirnames(1)
	// If no error, it means the directory was not empty.
	// If another error other than io.EOF, it means we
	// couldn't determine if the directory was empty.
	// (This happens if, for example, the path is not a directory.)
	return err == io.EOF
}
