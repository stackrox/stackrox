package snapshot

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

// Snapshot contains a Blob with read-only blob data backed by a temp file.
// The temp file will be removed on close.
type Snapshot interface {
	io.Reader
	io.Closer
	io.Seeker
	Name() string

	GetBlob() *storage.Blob
}

// NewBlobSnapshot creates a temp dir with a file. The file and its temp dir will be removed on closure.
func NewBlobSnapshot(p string, blob *storage.Blob) (Snapshot, error) {
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	return &snapshot{File: file, blob: blob, path: p}, nil
}

type snapshot struct {
	*os.File
	blob *storage.Blob
	path string
}

// Close temp file and remove its temp dir.
func (f *snapshot) Close() error {
	err := f.File.Close()
	if removeErr := os.RemoveAll(f.path); err != nil {
		log.Errorf("failed to remove %q: %v", f.path, removeErr)
	}
	return err
}

func (f *snapshot) Name() string {
	name := path.Base(f.blob.GetName())
	// It is definitely not a good practice to have use empty string or all slashes in
	// Blob name, but here is the workaround in case that happens.
	if name == "." || name == "/" {
		return "noname"
	}
	return name
}

func (f *snapshot) GetBlob() *storage.Blob {
	// return f.blob.Clone()
	return f.blob
}
