package fileutils

import (
	"os"
	"path/filepath"
)

// DirectorySize walks a directory and returns it's size in bytes
func DirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(subpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		size += info.Size()
		return err
	})
	if err != nil && !os.IsNotExist(err) {
		return size, err
	}
	return size, nil
}
