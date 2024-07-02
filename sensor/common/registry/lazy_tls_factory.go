package registry

import (
	"errors"
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/registries/docker"
	dockerFactory "github.com/stackrox/rox/pkg/registries/docker"
	rhelFactory "github.com/stackrox/rox/pkg/registries/rhel"
	"github.com/stackrox/rox/pkg/registries/types"
)

var (
	lazyEligibleCreators = []types.CreatorWrapper{
		dockerFactory.CreatorWithoutRepoList,
		rhelFactory.CreatorWithoutRepoList,
	}
)

type lazyFactory struct {
	creators      map[string]types.Creator
	tlsCheckCache *tlsCheckCacheImpl
}

var _ registries.Factory = (*lazyFactory)(nil)

func newLazyFactory(tlsCheckCache *tlsCheckCacheImpl) registries.Factory {
	factory := &lazyFactory{
		creators:      make(map[string]types.Creator),
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
		return nil, fmt.Errorf("registry with type '%s' does not exist", source.GetType())
	}

	dockerConfig, err := extractDockerConfig(source)
	if err != nil {
		// Only integrations with a docker config are eligible for lazy tls checking
		return nil, fmt.Errorf("extracting docker config: %w", err)
	}

	url := docker.FormatURL(dockerConfig.GetEndpoint())
	hostname := docker.RegistryServer(dockerConfig.GetEndpoint(), url)
	if hostname == "" {
		return nil, errors.New("empty registry host")
	}

	return &lazyTLSCheckRegistry{
		source:           source,
		creator:          creator,
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

func extractDockerConfig(ii *storage.ImageIntegration) (*storage.DockerConfig, error) {
	protoCfg := ii.GetIntegrationConfig()
	if protoCfg == nil {
		return nil, errors.New("image integration config is nil")
	}

	cfg, ok := protoCfg.(*storage.ImageIntegration_Docker)
	if !ok || cfg == nil {
		return nil, errors.New("image integration docker config is nil")
	}

	dockerConfig := cfg.Docker
	if dockerConfig == nil {
		return nil, errors.New("docker config is nil")
	}

	return dockerConfig, nil
}
