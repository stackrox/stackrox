package registry

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
)

var (
	// These are the only two possible registry types Sensor 'auto-creates'.
	// See `createImageIntegration` in sensor/common/registry/registry_store.go.
	lazyEligibleCreators = []types.CreatorWrapper{
		docker.CreatorWithoutRepoList,
		rhel.CreatorWithoutRepoList,
	}
)

var _ registries.Factory = (*lazyFactory)(nil)

type lazyFactory struct {
	creators      map[string]types.Creator
	tlsCheckCache *tlsCheckCacheImpl
}

func newLazyFactory(tlsCheckCache *tlsCheckCacheImpl) registries.Factory {
	factory := &lazyFactory{
		creators:      make(map[string]types.Creator, len(lazyEligibleCreators)),
		tlsCheckCache: tlsCheckCache,
	}

	for _, creatorFunc := range lazyEligibleCreators {
		typ, creator := creatorFunc()
		factory.creators[typ] = creator
	}

	return factory
}

// CreateRegistry performs aggressive up front validation so that errors can
// be surfaced, otherwise errors may be lost during lazy initialization.
func (e *lazyFactory) CreateRegistry(source *storage.ImageIntegration, options ...types.CreatorOption) (types.ImageRegistry, error) {
	if source == nil {
		return nil, errors.New("image integration is nil")
	}

	creator, exists := e.creators[source.GetType()]
	if !exists {
		return nil, errors.Errorf("registry with type '%s' does not exist", source.GetType())
	}

	dockerConfig := source.GetDocker()
	if dockerConfig == nil {
		// Only integrations with a docker config are eligible for lazy tls checking
		return nil, errors.New("docker config is nil")
	}

	hostname, url := docker.RegistryHostnameURL(dockerConfig.GetEndpoint())
	if hostname == "" {
		return nil, errors.New("empty registry hostname")
	}

	return &lazyTLSCheckRegistry{
		source:           source,
		creator:          creator,
		creatorOptions:   options,
		dockerConfig:     dockerConfig,
		url:              url,
		registryHostname: hostname,
		dataSource: &storage.DataSource{
			Id:   source.GetId(),
			Name: source.GetName(),
		},
		tlsCheckCache: e.tlsCheckCache,
	}, nil
}
