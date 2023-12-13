package registries

import (
	"github.com/stackrox/rox/generated/storage"
	artifactoryFactory "github.com/stackrox/rox/pkg/registries/artifactory"
	artifactRegistryFactory "github.com/stackrox/rox/pkg/registries/artifactregistry"
	azureFactory "github.com/stackrox/rox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	ecrFactory "github.com/stackrox/rox/pkg/registries/ecr"
	googleFactory "github.com/stackrox/rox/pkg/registries/google"
	ibmFactory "github.com/stackrox/rox/pkg/registries/ibm"
	nexusFactory "github.com/stackrox/rox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/rox/pkg/registries/quay"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"

	"github.com/stackrox/rox/pkg/registries/types"
)

// Creator is the func stub that defines how to instantiate an image registry.
type Creator func(scanner *storage.ImageIntegration) (types.Registry, error)

// Factory provides a centralized location for creating a Scanner from a v1.ImageIntegrations.
//
//go:generate mockgen-wrapper
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

// AllCreatorFuncsWithoutRepoList defines all known registry creators with repo list disabled.
var AllCreatorFuncsWithoutRepoList = []CreatorWrapper{
	artifactRegistryFactory.CreatorWithoutRepoList,
	artifactoryFactory.CreatorWithoutRepoList,
	dockerFactory.CreatorWithoutRepoList,
	ecrFactory.CreatorWithoutRepoList,
	googleFactory.CreatorWithoutRepoList,
	quayFactory.CreatorWithoutRepoList,
	nexusFactory.CreatorWithoutRepoList,
	azureFactory.CreatorWithoutRepoList,
	rhelFactory.CreatorWithoutRepoList,
	ibmFactory.CreatorWithoutRepoList,
}

// NewFactory creates a new registries factory.
func NewFactory(opts FactoryOptions) Factory {
	reg := &factoryImpl{
		creators:                make(map[string]Creator),
		creatorsWithoutRepoList: make(map[string]Creator),
	}

	creatorFuncs := AllCreatorFuncs
	if len(opts.CreatorFuncs) > 0 {
		creatorFuncs = opts.CreatorFuncs
	}

	for _, creatorFunc := range creatorFuncs {
		typ, creator := creatorFunc()
		reg.creators[typ] = creator
	}

	for _, creatorFunc := range opts.CreatorFuncsWithoutRepoList {
		typ, creator := creatorFunc()
		reg.creatorsWithoutRepoList[typ] = creator
	}

	return reg
}
