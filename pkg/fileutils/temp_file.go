package fileutils

import (
	"os"
	"path"
)

// CreateTempFile creates a temp dir with a file. The file and its temp dir will be removed on closure.
func CreateTempFile(p string, name string) (*tempFile, error) {
	file, err := os.Open(path.Join(p, name))
	if err != nil {
		return nil, err
	}
	return &tempFile{File: file, path: p, name: name}, nil
}

type tempFile struct {
	*os.File
	path string
	name string
}

// Close temp file and remove its temp dir.
func (f *tempFile) Close() error {
	err := f.File.Close()
	if removeErr := os.RemoveAll(f.path); err != nil {
		log.Errorf("failed to remove %q: %v", f.path, removeErr)
	}
	return err
}

func (f *tempFile) Name() string {
	return f.name
}
