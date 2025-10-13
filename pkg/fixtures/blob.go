package fixtures

import (
	"bytes"

	"github.com/stackrox/rox/generated/storage"
)

// GetBlobWithData returns a mocked blob with data for testing
func GetBlobWithData() (*storage.Blob, *bytes.Buffer) {
	buffer := bytes.NewBuffer([]byte("some data"))
	return &storage.Blob{
		Name:         "/blobpath",
		Oid:          123,
		Length:       int64(buffer.Len()),
		ModifiedTime: nil,
	}, buffer
}
