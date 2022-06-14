package registries

import (
	"github.com/stackrox/stackrox/generated/storage"
	artifactoryFactory "github.com/stackrox/stackrox/pkg/registries/artifactory"
	artifactRegistryFactory "github.com/stackrox/stackrox/pkg/registries/artifactregistry"
	azureFactory "github.com/stackrox/stackrox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/stackrox/pkg/registries/docker"
	ecrFactory "github.com/stackrox/stackrox/pkg/registries/ecr"
	googleFactory "github.com/stackrox/stackrox/pkg/registries/google"
	ibmFactory "github.com/stackrox/stackrox/pkg/registries/ibm"
	nexusFactory "github.com/stackrox/stackrox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/stackrox/pkg/registries/quay"
	rhelFactory "github.com/stackrox/stackrox/pkg/registries/rhel"

	"github.com/stackrox/stackrox/pkg/registries/types"
)

// Creator is the func stub that defines how to instantiate an image registry.
type Creator func(scanner *storage.ImageIntegration) (types.Registry, error)

// Factory provides a centralized location for creating a Scanner from a v1.ImageIntegrations.
type Factory interface {
	CreateRegistry(source *storage.ImageIntegration) (types.ImageRegistry, error)
}

// CreatorWrapper is a wrapper around a Creator which also returns the registry's name.
type CreatorWrapper func() (string, func(integration *storage.ImageIntegration) (types.Registry, error))

// AllCreatorFuncs defines all known registry creators.
var AllCreatorFuncs = []CreatorWrapper{
	artifactRegistryFactory.Creator,
	artifactoryFactory.Creator,
	dockerFactory.Creator,
	ecrFactory.Creator,
	googleFactory.Creator,
	quayFactory.Creator,
	nexusFactory.Creator,
	azureFactory.Creator,
	rhelFactory.Creator,
	ibmFactory.Creator,
}

// NewFactory creates a new scanner factory.
func NewFactory(opts FactoryOptions) Factory {
	reg := &factoryImpl{
		creators: make(map[string]Creator),
	}

	creatorFuncs := AllCreatorFuncs
	if len(opts.CreatorFuncs) > 0 {
		creatorFuncs = opts.CreatorFuncs
	}

	for _, creatorFunc := range creatorFuncs {
		typ, creator := creatorFunc()
		reg.creators[typ] = creator

	}

	return reg
}
