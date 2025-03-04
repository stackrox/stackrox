package docker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/docker/distribution/manifest/manifestlist"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
	pkgUtils "github.com/stackrox/rox/pkg/utils"
)

const (
	repoListInterval = 10 * time.Minute
)

var log = logging.LoggerForModule()

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, types.Creator) {
	return types.DockerType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := NewDockerRegistry(integration, false, cfg.GetMetricsHandler())
			return reg, err
		}
}

// CreatorWithoutRepoList provides the type and registries.Creator to add to the registries Registry.
// Populating the internal repo list will be disabled.
func CreatorWithoutRepoList() (string, types.Creator) {
	return types.DockerType,
		func(integration *storage.ImageIntegration, options ...types.CreatorOption) (types.Registry, error) {
			cfg := types.ApplyCreatorOptions(options...)
			reg, err := NewDockerRegistry(integration, true, cfg.GetMetricsHandler())
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

	repoListOnce sync.Once
}

// NewDockerRegistryWithConfig creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistryWithConfig(cfg *Config, integration *storage.ImageIntegration,
	transports ...registry.Transport,
) (*Registry, error) {
	hostname, url := RegistryHostnameURL(cfg.Endpoint)

	var transport registry.Transport
	if len(transports) == 0 || transports[0] == nil {
		transport = DefaultTransport(cfg)
	} else {
		transport = transports[0]
	}
	client, err := registry.NewFromTransport(url, transport, registry.Quiet)
	if err != nil {
		return nil, err
	}

	client.Client.Timeout = env.RegistryClientTimeout.DurationSetting()

	repoListState := pkgUtils.IfThenElse(cfg.DisableRepoList, "disabled", "enabled")
	log.Debugf("created integration %q with repo list %s", integration.GetName(), repoListState)

	return &Registry{
		url:                   url,
		registry:              hostname,
		Client:                client,
		cfg:                   cfg,
		protoImageIntegration: integration,
	}, nil
}

// NewDockerRegistry creates a generic docker registry integration
func NewDockerRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*Registry, error) {
	dockerConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Docker)
	if !ok {
		return nil, errors.New("Docker configuration required")
	}
	cfg := &Config{
		Endpoint:        dockerConfig.Docker.GetEndpoint(),
		username:        dockerConfig.Docker.GetUsername(),
		password:        dockerConfig.Docker.GetPassword(),
		Insecure:        dockerConfig.Docker.GetInsecure(),
		DisableRepoList: disableRepoList,
		MetricsHandler:  metricsHandler,
		RegistryType:    integration.GetType(),
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

	r.lazyLoadRepoList()

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

// lazyLoadRepoList will perform the initial repo list load if necessary.
// This is safe to call multiple times.
func (r *Registry) lazyLoadRepoList() {
	r.repoListOnce.Do(func() {
		repoSet, err := retrieveRepositoryList(r.Client)
		if err != nil {
			// This is not a critical error, matching will instead be performed solely
			// based on the registry endpoint (instead of endpoint AND repo list).
			log.Debugf("could not initialize repo list for integration %s: %v", r.protoImageIntegration.GetName(), err)
			return
		}

		r.repositoryList = repoSet
		r.repositoryListTicker = time.NewTicker(repoListInterval)
	})
}

func handleManifests(r *Registry, manifestType, remote string, dig digest.Digest) (*storage.ImageMetadata, error) {
	if err := dig.Validate(); err != nil {
		return nil, errors.Wrap(err, "invalid image digest")
	}

	ref := dig.String()

	// Note: Any updates here must be accompanied by updates to registry_without_digest.go.
	switch manifestType {
	case manifestV1.MediaTypeManifest:
		return HandleV1Manifest(r, remote, ref)
	case manifestV1.MediaTypeSignedManifest:
		return HandleV1SignedManifest(r, remote, ref)
	case manifestlist.MediaTypeManifestList:
		return HandleV2ManifestList(r, remote, ref)
	case manifestV2.MediaTypeManifest:
		return HandleV2Manifest(r, remote, ref)
	case registry.MediaTypeImageIndex:
		return HandleOCIImageIndex(r, remote, ref)
	case registry.MediaTypeImageManifest:
		return HandleOCIManifest(r, remote, ref)
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
	d, manifestType, err := r.Client.ManifestDigest(remote, utils.Reference(image))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the manifest digest")
	}
	if err := d.Validate(); err != nil {
		return nil, errors.Wrap(err, "digest is invalid")
	}
	return handleManifests(r, manifestType, remote, d)
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
func (r *Registry) Config(_ context.Context) *types.Config {
	username, password := r.cfg.GetCredentials()
	return &types.Config{
		Username:         username,
		Password:         password,
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
