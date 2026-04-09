package registry

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Provider is a read-only interface for accessing registry information.
type Provider interface {
	GetPullSecretRegistries(image *storage.ImageName, namespace string, imagePullSecrets []string) ([]types.ImageRegistry, error)
	GetGlobalRegistries(*storage.ImageName) ([]types.ImageRegistry, error)
	GetCentralRegistries(*storage.ImageName) []types.ImageRegistry
	IsLocal(*storage.ImageName) bool
}
