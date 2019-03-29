package zip

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

// Wrapper is a wrapper around the current zip implementation so we can
// also output to files if we have direct filesystem access
type Wrapper struct {
	content map[string]*File
}

// NewWrapper creates a new zip file wrapper
func NewWrapper() *Wrapper {
	return &Wrapper{
		content: make(map[string]*File),
	}
}

// AddFiles adds files to the wrapper
func (w *Wrapper) AddFiles(files ...*File) {
	for _, f := range files {
		w.content[f.Name] = f
	}
}

// getFilePerms returns the appropriate Unix file permissions for the given
// file manifest.
func getFilePerms(f *File) os.FileMode {
	var (
		isExecutable = f.Flags&Executable != 0
		isSensitive  = f.Flags&Sensitive != 0
	)
	switch {
	case isExecutable && isSensitive:
		// u  |g  |o
		// rwx|---|---
		return os.FileMode(0700)
	case isExecutable:
		// u  |g  |o
		// rwx|r-x|r-x
		return os.FileMode(0755)
	case isSensitive:
		// u  |g  |o
		// rw-|---|---
		return os.FileMode(0600)
	default:
		// u  |g  |o
		// rw-|r--|r--
		return os.FileMode(0644)
	}
}

// Zip returns the bytes of the zip archive or an error
func (w *Wrapper) Zip() ([]byte, error) {
	var b []byte
	buf := bytes.NewBuffer(b)
	zipW := zip.NewWriter(buf)
	for _, file := range w.content {
		hdr := &zip.FileHeader{
			Name: file.Name,
		}
		hdr.Modified = time.Now()
		hdr.SetMode(getFilePerms(file))

		f, err := zipW.CreateHeader(hdr)
		if err != nil {
			return nil, errors.Wrap(err, "file creation")
		}
		_, err = f.Write(file.Content)
		if err != nil {
			return nil, errors.Wrap(err, "file writing")
		}
	}
	if err := zipW.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Directory writes the contents to the passed directory on the filesystem
func (w *Wrapper) Directory(path string) (string, error) {
	if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return "", err
	} else if err == nil {
		return "", fmt.Errorf("Directory %q already exists. Please specify and new path to ensure expected results", path)
	}

	for _, file := range w.content {
		fullDirPath := filepath.Join(path, filepath.Dir(file.Name))
		if err := os.MkdirAll(fullDirPath, 0755); err != nil {
			return "", err
		}
		fullFilePath := filepath.Join(path, file.Name)
		if err := ioutil.WriteFile(fullFilePath, file.Content, getFilePerms(file)); err != nil {
			return "", err
		}
	}
	return path, nil
}
