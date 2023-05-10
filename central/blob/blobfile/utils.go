package blobfile

import (
	"context"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/blob/datastore"
	"github.com/stackrox/rox/generated/storage"
)

func BlobSnapshot(blobStore datastore.Datastore, name string) (rnc ReadOnlyBlobFile, err error) {
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

	tempFile := filepath.Join(tempDir, "blob.data")
	var writer *os.File
	writer, err = os.Create(tempFile)
	if err != nil {
		return
	}

	var blob *storage.Blob
	blob, _, err = blobStore.Get(context.Background(), name, writer)
	if err != nil {
		err = errors.Wrapf(err, "failed to open blob with name %q", name)
		return
	}
	err = writer.Close()
	if err != nil {
		return
	}
	rnc, err = CreateBlobFile(tempFile, blob)
	if err != nil {
		return
	}
	return
}
