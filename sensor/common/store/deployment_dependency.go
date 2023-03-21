package store

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/sensor/common/service"
)

// ImageMetadata holds the pull details of an image which is required by local scanning.
type ImageMetadata struct {
	NotPullable    bool
	IsClusterLocal bool
}

// Dependencies are properties that belong to a storage.Deployment object, but don't come directly from the
// k8s deployment spec. They need to be enhanced from other resources, like RBACs and Services.
type Dependencies struct {
	PermissionLevel storage.PermissionLevel
	Exposures       []map[service.PortRef][]*storage.PortConfig_ExposureInfo

	// ImageMetadata refers to the images in a deployment that might be in a Local Cluster. This is needed in case the
	// secrets change and deployment containers need to be reprocessed.
	ImageMetadata map[string]ImageMetadata
}
