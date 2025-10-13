package snapshot

import (
	"context"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()

	// ErrBlobNotExist is Blob does not exist error
	ErrBlobNotExist = errors.New("cannot find blob")
)

// Snapshot contains a Blob with its data backed by a temp file.
// The temp file will be removed on close.
type Snapshot struct {
	*os.File
	blob     *storage.Blob
	tmpDir   string
	baseName string
}

// Close temp file and remove its temp dir.
func (s *Snapshot) Close() error {
	if s == nil || s.File == nil {
		return nil
	}
	err := s.File.Close()
	if removeErr := os.RemoveAll(s.tmpDir); removeErr != nil {
		log.Errorf("failed to remove %q: %v", s.tmpDir, removeErr)
	}
	return err
}

// GetBlob returns Blob
func (s *Snapshot) GetBlob() *storage.Blob {
	return s.blob
}

// TakeBlobSnapshot create a Snapshot for the named blob if it exists
func TakeBlobSnapshot(ctx context.Context, blobStore datastore.Datastore, name string) (rnc *Snapshot, err error) {
	var tempDir string
	tempDir, err = os.MkdirTemp("", "blob-file-")
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tempDir)
		}
	}()

	baseName := path.Base(name)
	// It is definitely not a good practice to have use empty string or all slashes in
	// Blob name, but here is the workaround in case that happens.
	if baseName == "." || baseName == "/" {
		baseName = "noname"
	}
	tempFile := filepath.Join(tempDir, baseName)

	var writer *os.File
	if writer, err = os.Create(tempFile); err != nil {
		return
	}

	var blob *storage.Blob
	var exists bool
	if blob, exists, err = blobStore.Get(ctx, name, writer); err != nil {
		err = errors.Wrapf(err, "failed to open blob with name %q", name)
		return
	}
	if err = writer.Close(); err != nil {
		return
	}
	if !exists {
		err = ErrBlobNotExist
		return
	}

	file, err := os.Open(tempFile)
	if err != nil {
		return nil, err
	}
	return &Snapshot{File: file, blob: blob, tmpDir: tempDir, baseName: baseName}, nil
}
