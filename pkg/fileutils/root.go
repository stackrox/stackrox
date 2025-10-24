package fileutils

import (
	"os"
	"path/filepath"
)

// MkdirAllInRoot creates all directories in the path recursively within the secured root.
// This is a helper for os.Root which doesn't have a MkdirAll method.
// It ensures all directory creation happens within the constraints of the os.Root,
// providing protection against path traversal attacks.
func MkdirAllInRoot(root *os.Root, dirPath string, perm os.FileMode) error {
	if dirPath == "." || dirPath == "" {
		return nil
	}

	// Clean the path and split into components
	dirPath = filepath.Clean(dirPath)
	if dirPath == "." {
		return nil
	}

	// Try to create the directory - if it fails because parent doesn't exist, create parent first
	err := root.Mkdir(dirPath, perm)
	if err == nil || os.IsExist(err) {
		return nil
	}

	// Create parent directory first
	parent := filepath.Dir(dirPath)
	if parent != "." && parent != dirPath {
		if err := MkdirAllInRoot(root, parent, perm); err != nil {
			return err
		}
		// Try again after creating parent
		err = root.Mkdir(dirPath, perm)
		if err == nil || os.IsExist(err) {
			return nil
		}
	}

	return err
}
