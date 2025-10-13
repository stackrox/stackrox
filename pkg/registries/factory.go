package registries

import (
	"github.com/stackrox/rox/generated/storage"
	artifactoryFactory "github.com/stackrox/rox/pkg/registries/artifactory"
	artifactRegistryFactory "github.com/stackrox/rox/pkg/registries/artifactregistry"
	azureFactory "github.com/stackrox/rox/pkg/registries/azure"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	ecrFactory "github.com/stackrox/rox/pkg/registries/ecr"
	ghcrFactory "github.com/stackrox/rox/pkg/registries/ghcr"
	googleFactory "github.com/stackrox/rox/pkg/registries/google"
	ibmFactory "github.com/stackrox/rox/pkg/registries/ibm"
	nexusFactory "github.com/stackrox/rox/pkg/registries/nexus"
	quayFactory "github.com/stackrox/rox/pkg/registries/quay"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Factory provides a centralized location for creating a Scanner from a v1.ImageIntegrations.
//
//go:generate mockgen-wrapper
type Factory interface {
	CreateRegistry(source *storage.ImageIntegration, options ...types.CreatorOption) (types.ImageRegistry, error)
}

// AllCreatorFuncs defines all known registry creators.
var AllCreatorFuncs = []types.CreatorWrapper{
	artifactRegistryFactory.Creator,
	artifactoryFactory.Creator,
	azureFactory.Creator,
	dockerFactory.Creator,
	ecrFactory.Creator,
	ghcrFactory.Creator,
	googleFactory.Creator,
	ibmFactory.Creator,
	nexusFactory.Creator,
	quayFactory.Creator,
	rhelFactory.Creator,
}

// AllCreatorFuncsWithoutRepoList defines all known registry creators with repo list disabled.
var AllCreatorFuncsWithoutRepoList = []types.CreatorWrapper{
	artifactRegistryFactory.CreatorWithoutRepoList,
	artifactoryFactory.CreatorWithoutRepoList,
	azureFactory.CreatorWithoutRepoList,
	dockerFactory.CreatorWithoutRepoList,
	ecrFactory.CreatorWithoutRepoList,
	ghcrFactory.CreatorWithoutRepoList,
	googleFactory.CreatorWithoutRepoList,
	ibmFactory.CreatorWithoutRepoList,
	nexusFactory.CreatorWithoutRepoList,
	quayFactory.CreatorWithoutRepoList,
	rhelFactory.CreatorWithoutRepoList,
}

// NewFactory creates a new registries factory.
func NewFactory(opts FactoryOptions) Factory {
	reg := &factoryImpl{
		creators:                make(map[string]types.Creator),
		creatorsWithoutRepoList: make(map[string]types.Creator),
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
