package manager

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
)

// Manager implements business logic related to collector probe uploads.
type Manager interface {
	Initialize() error
	GetExistingProbeFiles(ctx context.Context, files []string) ([]*v1.ProbeUploadManifest_File, error)
}
