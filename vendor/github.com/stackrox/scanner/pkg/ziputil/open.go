package ziputil

import (
	"archive/zip"
	"io"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ReadCloser is a wrapper around io.ReadCloser for reading files in a ZIP.
type ReadCloser struct {
	io.ReadCloser
	Name string
}

// OpenFile opens the given file in the zip, and returns the ReadCloser.
// It returns an error if the file was not found, or if there was an error opening
// the file.
func OpenFile(zipR *zip.ReadCloser, name string) (*ReadCloser, error) {
	for _, file := range zipR.File {
		if file.Name == name {
			f, err := file.Open()
			if err != nil {
				return nil, err
			}

			return &ReadCloser{
				ReadCloser: f,
				Name:       name,
			}, nil
		}
	}
	return nil, errors.Errorf("file %q not found in zip", name)
}

// OpenFilesInDir opens the files with the given suffix which are in the given dir.
// It returns an error if any of the files cannot be opened.
func OpenFilesInDir(zipR *zip.ReadCloser, dir string, suffix string) ([]*ReadCloser, error) {
	var rs []*ReadCloser
	for _, file := range zipR.File {
		if within(dir, file.Name) && strings.HasSuffix(file.Name, suffix) {
			f, err := file.Open()
			if err != nil {
				return nil, errors.Wrapf(err, "unable to open file %s in directory %s", file.Name, dir)
			}
			rs = append(rs, &ReadCloser{
				ReadCloser: f,
				Name:       file.Name,
			})
		}
	}

	return rs, nil
}

// within returns true if sub is within the parent.
// This function is inspired by https://github.com/mholt/archiver/blob/v3.5.0/archiver.go#L360
func within(parent, sub string) bool {
	rel, err := filepath.Rel(parent, sub)
	if err != nil {
		return false
	}
	return !strings.Contains(rel, "..")
}
