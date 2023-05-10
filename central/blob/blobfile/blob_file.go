package blobfile

import (
	"io"
	"os"
	"path"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

type ReadOnlyBlobFile interface {
	io.Reader
	io.Closer
	io.Seeker
	Name() string

	GetBlob() *storage.Blob
}

// CreateBlobFile creates a temp dir with a file. The file and its temp dir will be removed on closure.
func CreateBlobFile(p string, blob *storage.Blob) (*blobFile, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return &blobFile{File: file, blob: blob, path: p}, nil
}

type blobFile struct {
	*os.File
	blob *storage.Blob
	path string
}

// Close temp file and remove its temp dir.
func (f *blobFile) Close() error {
	err := f.File.Close()
	if removeErr := os.RemoveAll(f.path); err != nil {
		log.Errorf("failed to remove %q: %v", f.path, removeErr)
	}
	return err
}

func (f *blobFile) Name() string {
	name := path.Base(f.blob.GetName())
	if name == "." || name == "/" {
		return "noname"
	}
	return name
}

func (f *blobFile) GetBlob() *storage.Blob {
	// return f.blob.Clone()
	return f.blob
}
