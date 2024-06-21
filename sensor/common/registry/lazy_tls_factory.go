package registry

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
)

var (
	lazyEligableIntegrationTypes = set.NewFrozenSet(
		types.DockerType,
		types.RedHatType,
	)
)

type LazyFactory struct {
	creators       map[string]types.Creator
	defaultFactory registries.Factory
	tlsCheckCache  *tlsCheckCacheImpl
}

var _ registries.Factory = (*LazyFactory)(nil)

func NewLazyFactory(defaultFactory registries.Factory, tlsCheckCache *tlsCheckCacheImpl) registries.Factory {
	factory := &LazyFactory{
		creators:       make(map[string]types.Creator),
		defaultFactory: defaultFactory,
		tlsCheckCache:  tlsCheckCache,
	}

	for _, creatorFunc := range registries.AllCreatorFuncsWithoutRepoList {
		typ, creator := creatorFunc()
		factory.creators[typ] = creator
	}

	return factory
}

func (e *LazyFactory) CreateRegistry(source *storage.ImageIntegration, options ...types.CreatorOption) (types.ImageRegistry, error) {
	if !lazyEligableIntegrationTypes.Contains(source.GetType()) {
		return e.defaultFactory.CreateRegistry(source, options...)
	}

	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, fmt.Errorf("registry with type '%s' does not exist", source.GetType())
	}

	return &LazyTLSCheckRegistry{
		source:  source,
		creator: creator,
		dataSource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
		tlsCheckCache: e.tlsCheckCache,
	}, nil
}
