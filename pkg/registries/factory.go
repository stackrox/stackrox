package registries

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries/types"
)

// Factory provides a centralized location for creating a Scanner from a v1.ImageIntegrations.
//
//go:generate mockgen-wrapper
type Factory interface {
	CreateRegistry(source *storage.ImageIntegration, options ...types.CreatorOption) (types.ImageRegistry, error)
}

// NewFactory creates a new registries factory.
// Callers must provide the desired creator functions via FactoryOptions.
// For all registry types, use registries/all.CreatorFuncs.
func NewFactory(opts FactoryOptions) Factory {
	reg := &factoryImpl{
		creators:                make(map[string]types.Creator),
		creatorsWithoutRepoList: make(map[string]types.Creator),
	}

	for _, creatorFunc := range opts.CreatorFuncs {
		typ, creator := creatorFunc()
		reg.creators[typ] = creator
	}

	for _, creatorFunc := range opts.CreatorFuncsWithoutRepoList {
		typ, creator := creatorFunc()
		reg.creatorsWithoutRepoList[typ] = creator
	}

	return reg
}
