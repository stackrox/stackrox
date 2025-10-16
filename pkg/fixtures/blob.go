package fixtures

import (
	"bytes"

	"github.com/stackrox/rox/generated/storage"
)

// GetBlobWithData returns a mocked blob with data for testing
func GetBlobWithData() (*storage.Blob, *bytes.Buffer) {
	buffer := bytes.NewBuffer([]byte("some data"))
	blob := &storage.Blob{}
	blob.SetName("/blobpath")
	blob.SetOid(123)
	blob.SetLength(int64(buffer.Len()))
	blob.ClearModifiedTime()
	return blob, buffer
}
