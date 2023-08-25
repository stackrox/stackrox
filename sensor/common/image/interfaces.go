package image

import (
	"context"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/sensor/common/scan"
	"google.golang.org/grpc"
)

//go:generate mockgen-wrapper
type registryStore interface {
	GetRegistryForImageInNamespace(*storage.ImageName, string) (registryTypes.ImageRegistry, error)
	GetGlobalRegistryForImage(*storage.ImageName) (registryTypes.ImageRegistry, error)
	GetMatchingCentralRegistryIntegrations(*storage.ImageName) []registryTypes.ImageRegistry
	IsLocal(*storage.ImageName) bool
}

type centralClient interface {
	ScanImageInternal(context.Context, *v1.ScanImageInternalRequest, ...grpc.CallOption) (*v1.ScanImageInternalResponse, error)
	EnrichLocalImageInternal(context.Context, *v1.EnrichLocalImageInternalRequest, ...grpc.CallOption) (*v1.ScanImageInternalResponse, error)
}

type localScan interface {
	EnrichLocalImageInNamespace(context.Context, scan.LocalScanCentralClient, *storage.ContainerImage, string, string, bool) (*storage.Image, error)
}
