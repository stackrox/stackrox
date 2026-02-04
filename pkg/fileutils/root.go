package fileutils

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// MkdirAllInRoot creates all directories in the path iteratively within the secured root.
// This is a helper for os.Root which doesn't have a MkdirAll method.
// It ensures all directory creation happens within the constraints of the os.Root,
// providing protection against path traversal attacks.
func MkdirAllInRoot(root *os.Root, dirPath string, perm os.FileMode) error {
	// Clean the path - os.Root will reject absolute paths and traversals
	cleanPath := filepath.Clean(dirPath)
	if cleanPath == "." || cleanPath == "" {
		return nil
	}

	// For absolute paths, let os.Root reject them
	if filepath.IsAbs(cleanPath) {
		return root.Mkdir(cleanPath, perm)
	}

	// Iteratively build each segment under root for relative paths
	var current string
	for _, seg := range strings.Split(cleanPath, string(filepath.Separator)) {
		current = filepath.Join(current, seg)
		if err := root.Mkdir(current, perm); err != nil && !os.IsExist(err) {
			return err
		}
	}
	return nil
}

// WriteFileInRoot sanitizes name, creates parent directories in root, and writes all content from rc.
// This helper combines directory creation, file opening, and content copying into a single safe operation.
// The ReadCloser is always closed, even on error.
func WriteFileInRoot(root *os.Root, name string, perm os.FileMode, rc io.ReadCloser) error {
	defer utils.IgnoreError(rc.Close)

	cleaned := filepath.Clean(name)
	dir := filepath.Dir(cleaned)
	if dir != "." && dir != "" {
		if err := MkdirAllInRoot(root, dir, 0o755); err != nil {
			return errors.Wrapf(err, "creating dir %q", dir)
		}
	}

	f, err := root.OpenFile(cleaned, os.O_CREATE|os.O_WRONLY|os.O_EXCL, perm)
	if err != nil {
		return errors.Wrapf(err, "opening file %q", cleaned)
	}
	defer utils.IgnoreError(f.Close)

	if _, err := io.Copy(f, rc); err != nil {
		return errors.Wrapf(err, "writing file %q", cleaned)
	}
	return nil
}
