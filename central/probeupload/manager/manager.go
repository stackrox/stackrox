package manager

import (
	"context"
	"io"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Manager implements business logic related to collector probe uploads.
type Manager interface {
	Initialize() error
	GetExistingProbeFiles(ctx context.Context, files []string) ([]*v1.ProbeUploadManifest_File, error)

	StoreFile(ctx context.Context, file string, data io.Reader, size int64, crc32 uint32) error

	// OpenFile attempts to open a probe file, returning the data reader and its size, or an error. This function does
	// not perform any access checks.
	LoadProbe(ctx context.Context, file string) (io.ReadCloser, int64, error)

	IsAvailable(ctx context.Context) (bool, error)
}
