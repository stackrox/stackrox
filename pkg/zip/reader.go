package zip

import (
	"archive/zip"
	"io"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// Reader opens a zip archive and reads from it.
type Reader struct {
	rc *zip.ReadCloser
}

// NewReader constructs a zip Reader from its file path.
func NewReader(path string) (*Reader, error) {
	rc, err := zip.OpenReader(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open zip archive %s for read.", path)
	}
	return &Reader{rc: rc}, nil
}

func (r *Reader) getFile(filePath string) *zip.File {
	for _, f := range r.rc.File {
		if f.Name == filePath {
			return f
		}
	}
	return nil
}

// ContainsFile tests and returns if the file exists.
func (r *Reader) ContainsFile(filePath string) bool {
	f := r.getFile(filePath)
	return f != nil
}

// ReadFrom gets the content of the file from zip archive.
func (r *Reader) ReadFrom(filePath string) ([]byte, error) {
	zipFile := r.getFile(filePath)
	if zipFile == nil {
		return nil, errors.Errorf("zip file does not contain %s", filePath)
	}

	f, err := zipFile.Open()
	if err != nil {
		return nil, errors.Wrapf(err, "unable to open %s", filePath)
	}
	defer utils.IgnoreError(f.Close)

	return io.ReadAll(f)
}

// Close closes the zip reader.
func (r *Reader) Close() error {
	return r.rc.Close()
}
