package docker

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/docker/distribution/manifest/manifestlist"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	registryTimeout  = 5 * time.Second
	repoListInterval = 10 * time.Minute
)

var log = logging.LoggerForModule()

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.DockerType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := NewDockerRegistry(integration, false)
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.DockerType,
		func(integration *storage.ImageIntegration, _ ...types.CreatorOption) (types.Registry, error) {
			reg, err := NewDockerRegistry(integration, true)
			return reg, err
		}
}

var _ types.Registry = (*Registry)(nil)

// Registry is the basic docker registry implementation
type Registry struct {
	cfg                   *Config
	protoImageIntegration *storage.ImageIntegration

	Client *registry.Registry

	url      string
	registry string // This is the registry portion of the image

	repositoryList       set.StringSet
	repositoryListTicker *time.Ticker
	repositoryListLock   sync.RWMutex
}

// NewDockerRegistryWithConfig creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistryWithConfig(cfg *Config, integration *storage.ImageIntegration) (*Registry, error) {
	url := cfg.formatURL()
	// if the registryServer endpoint contains docker.io then the image will be docker.io/namespace/repo:tag
	registryServer := urlfmt.GetServerFromURL(url)
	if strings.Contains(cfg.Endpoint, "docker.io") {
		registryServer = "docker.io"
	}

	client, err := registry.NewFromTransport(url, cfg.GetTransport(), registry.Quiet)
	if err != nil {
		return nil, err
	}

	client.Client.Timeout = registryTimeout

	var repoSet set.Set[string]
	var ticker *time.Ticker
	if !cfg.DisableRepoList {
		repoSet, err = retrieveRepositoryList(client)
		if err != nil {
			// This is not a critical error so it is purposefully not returned
			log.Debugf("could not update repo list for integration %s: %v", integration.GetName(), err)
		}
		ticker = time.NewTicker(repoListInterval)
		log.Debugf("created integration %q with repo list enabled", integration.GetName())
	} else {
		log.Debugf("created integration %q with repo list disabled", integration.GetName())
	}

	return &Registry{
		url:                   url,
		registry:              registryServer,
		Client:                client,
		cfg:                   cfg,
		protoImageIntegration: integration,

		repositoryList:       repoSet,
		repositoryListTicker: ticker,
	}, nil
}

// NewDockerRegistry creates a generic docker registry integration
func NewDockerRegistry(integration *storage.ImageIntegration, disableRepoList bool) (*Registry, error) {
	dockerConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Docker)
	if !ok {
		return nil, errors.New("Docker configuration required")
	}
	cfg := &Config{
		Endpoint:        dockerConfig.Docker.GetEndpoint(),
		Username:        dockerConfig.Docker.GetUsername(),
		Password:        dockerConfig.Docker.GetPassword(),
		Insecure:        dockerConfig.Docker.GetInsecure(),
		DisableRepoList: disableRepoList,
	}
	return NewDockerRegistryWithConfig(cfg, integration)
}

func retrieveRepositoryList(client *registry.Registry) (set.StringSet, error) {
	repos, err := client.Repositories()
	if err != nil {
		return nil, err
	}
	if len(repos) == 0 {
		return nil, errors.New("empty response from repositories call")
	}
	return set.NewStringSet(repos...), nil
}

// Match decides if the image is contained within this registry
func (r *Registry) Match(image *storage.ImageName) bool {
	match := urlfmt.TrimHTTPPrefixes(r.registry) == image.GetRegistry()
	if !match || r.cfg.DisableRepoList {
		return match
	}

	list := concurrency.WithRLock1(&r.repositoryListLock, func() set.StringSet {
		return r.repositoryList
	})
	if list == nil {
		return match
	}

	// Lazily update if the ticker has elapsed
	select {
	case <-r.repositoryListTicker.C:
		newRepoSet, err := retrieveRepositoryList(r.Client)
		if err != nil {
			log.Debugf("could not update repo list for integration %s: %v", r.protoImageIntegration.GetName(), err)
		} else {
			concurrency.WithLock(&r.repositoryListLock, func() {
				r.repositoryList = newRepoSet
			})
		}
	default:
	}

	r.repositoryListLock.RLock()
	defer r.repositoryListLock.RUnlock()

	return r.repositoryList.Contains(image.GetRemote())
}

func handleManifests(r *Registry, manifestType, remote, digest string) (*storage.ImageMetadata, error) {
	// Note: Any updates here must be accompanied by updates to registry_without_digest.go.
	switch manifestType {
	case manifestV1.MediaTypeManifest:
		return HandleV1Manifest(r, remote, digest)
	case manifestV1.MediaTypeSignedManifest:
		return HandleV1SignedManifest(r, remote, digest)
	case manifestlist.MediaTypeManifestList:
		return HandleV2ManifestList(r, remote, digest)
	case manifestV2.MediaTypeManifest:
		return HandleV2Manifest(r, remote, digest)
	case registry.MediaTypeImageIndex:
		return HandleOCIImageIndex(r, remote, digest)
	case registry.MediaTypeImageManifest:
		return HandleOCIManifest(r, remote, digest)
	default:
		return nil, fmt.Errorf("unknown manifest type '%s'", manifestType)
	}
}

// Metadata returns the metadata via this registry's implementation
func (r *Registry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if image == nil {
		return nil, nil
	}

	remote := image.GetName().GetRemote()
	digest, manifestType, err := r.Client.ManifestDigest(remote, utils.Reference(image))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the manifest digest")
	}
	return handleManifests(r, manifestType, remote, digest.String())
}

// Test tests the current registry and makes sure that it is working properly
func (r *Registry) Test() error {
	err := r.Client.Ping()
	if err != nil {
		log.Errorf("error testing docker integration: %v", err)
		if e, _ := err.(*registry.ClientError); e != nil {
			return errors.Errorf("error testing integration (code: %d). Please check Central logs for full error", e.Code())
		}
		return err
	}
	return nil
}

// Config returns the configuration of the docker registry
func (r *Registry) Config() *types.Config {
	return &types.Config{
		Username:         r.cfg.Username,
		Password:         r.cfg.Password,
		Insecure:         r.cfg.Insecure,
		URL:              r.url,
		RegistryHostname: r.registry,
		Autogenerated:    r.protoImageIntegration.GetAutogenerated(),
	}
}

// Name returns the name of the registry
func (r *Registry) Name() string {
	return r.protoImageIntegration.GetName()
}

// HTTPClient returns the *http.Client used to contact the registry.
func (r *Registry) HTTPClient() *http.Client {
	return r.Client.Client
}
