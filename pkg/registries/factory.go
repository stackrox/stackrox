package registries

import (
	"github.com/stackrox/rox/generated/storage"
	artifactoryFactory "github.com/stackrox/rox/pkg/registries/artifactory"
	artifactRegistryFactory "github.com/stackrox/rox/pkg/registries/artifactregistry"
	azureFactory "github.com/stackrox/rox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	dtrFactory "github.com/stackrox/rox/pkg/registries/dtr"
	ecrFactory "github.com/stackrox/rox/pkg/registries/ecr"
	googleFactory "github.com/stackrox/rox/pkg/registries/google"
	ibmFactory "github.com/stackrox/rox/pkg/registries/ibm"
	nexusFactory "github.com/stackrox/rox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/rox/pkg/registries/quay"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	tenableFactory "github.com/stackrox/rox/pkg/registries/tenable"

	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator is the func stub that defines how to instantiate an image registry.
type Creator func(scanner *storage.ImageIntegration) (types.Registry, error)

// Factory provides a centralized location for creating a Scanner from a v1.ImageIntegrations.
type Factory interface {
	CreateRegistry(source *storage.ImageIntegration) (types.ImageRegistry, error)
}

type creatorWrapper func() (string, func(integration *storage.ImageIntegration) (types.Registry, error))

var allCreatorFuncs = []creatorWrapper{
	artifactRegistryFactory.Creator,
	artifactoryFactory.Creator,
	dockerFactory.Creator,
	dtrFactory.Creator,
	ecrFactory.Creator,
	googleFactory.Creator,
	quayFactory.Creator,
	tenableFactory.Creator,
	nexusFactory.Creator,
	azureFactory.Creator,
	rhelFactory.Creator,
	ibmFactory.Creator,
}

// NewFactory creates a new scanner factory.
func NewFactory(opts ...FactoryOption) Factory {
	var o factoryOption
	for _, opt := range opts {
		opt.apply(&o)
	}

	reg := &factoryImpl{
		creators: make(map[string]Creator),
	}

	creatorFuncs := allCreatorFuncs
	if len(o.creatorFuncs) > 0 {
		creatorFuncs = o.creatorFuncs
	}

	for _, creatorFunc := range creatorFuncs {
		typ, creator := creatorFunc()
		reg.creators[typ] = creator

	}

	return reg
}
