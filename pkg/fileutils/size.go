package fileutils

import (
	"os"
	"path/filepath"
)

// DirectorySize walks a directory and returns its size in bytes
func DirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.WalkDir(path, func(subpath string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		size += info.Size()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		return size, err
	}
	return size, nil
}
