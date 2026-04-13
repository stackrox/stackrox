package docker

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"encoding/json"
	"io"

	"github.com/docker/distribution/manifest/manifestlist"
	manifestV1 "github.com/docker/distribution/manifest/schema1"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
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

	client *registryClient

	url       string
	registry  string // This is the registry portion of the image
	transport http.RoundTripper

	repositoryList       set.StringSet
	repositoryListTicker *time.Ticker
	repositoryListLock   sync.RWMutex

	repoListOnce sync.Once

	clientTimeout time.Duration
}

// NewDockerRegistryWithConfig creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistryWithConfig(cfg *Config, integration *storage.ImageIntegration,
	transports ...http.RoundTripper,
) (*Registry, error) {
	hostname, url := RegistryHostnameURL(cfg.Endpoint)

	var transport http.RoundTripper
	if len(transports) == 0 || transports[0] == nil {
		transport = DefaultTransport(cfg)
	} else {
		transport = transports[0]
	}

	username, password := cfg.GetCredentials()
	client := newRegistryClient(url, username, password, transport)

	repoListState := pkgUtils.IfThenElse(cfg.DisableRepoList, "disabled", "enabled")
	log.Debugf("created integration %q with repo list %s", integration.GetName(), repoListState)
	r := &Registry{
		url:                   url,
		registry:              hostname,
		client:                client,
		transport:             transport,
		cfg:                   cfg,
		protoImageIntegration: integration,
		clientTimeout:         env.RegistryClientTimeout.DurationSetting(),
	}
	return r, nil
}

// NewDockerRegistry creates a generic docker registry integration
func NewDockerRegistry(integration *storage.ImageIntegration, disableRepoList bool,
	metricsHandler *types.MetricsHandler,
) (*Registry, error) {
	dockerConfig, ok := integration.GetIntegrationConfig().(*storage.ImageIntegration_Docker)
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

func (r *Registry) retrieveRepositoryList() (set.StringSet, error) {
	repos, err := r.client.repositories(context.Background())
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
		newRepoSet, err := r.retrieveRepositoryList()
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
		repoSet, err := r.retrieveRepositoryList()
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
	case MediaTypeImageIndex:
		return HandleOCIImageIndex(r, remote, digest)
	case MediaTypeImageManifest:
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
	digest, manifestType, err := r.client.manifestDigest(context.Background(), remote, utils.Reference(image))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get the manifest digest")
	}
	return handleManifests(r, manifestType, remote, digest)
}

// Test tests the current registry and makes sure that it is working properly
func (r *Registry) Test() error {
	err := r.client.ping(context.Background())
	if err != nil {
		log.Errorf("error testing docker integration: %v", err)
		if e, _ := err.(*registryClientError); e != nil {
			return errors.Errorf("error testing integration (code: %d). Please check Central logs for full error", e.StatusCode)
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
	return &http.Client{
		Transport: r.transport,
		Timeout:   r.clientTimeout,
	}
}

// buildTransport builds an http.RoundTripper with timeouts, TLS settings, and
// metrics configured from the registry config. This transport is suitable for
// use with go-containerregistry's remote.WithAuth() for authentication.
func (r *Registry) buildTransport() http.RoundTripper {
	transport := proxy.RoundTripper(
		proxy.WithDialTimeout(env.RegistryDialerTimeout.DurationSetting()),
		proxy.WithResponseHeaderTimeout(env.RegistryResponseTimeout.DurationSetting()),
	)
	if r.cfg.Insecure {
		transport = proxy.RoundTripper(
			proxy.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
			proxy.WithDialTimeout(env.RegistryDialerTimeout.DurationSetting()),
			proxy.WithResponseHeaderTimeout(env.RegistryResponseTimeout.DurationSetting()),
		)
	}
	return r.cfg.MetricsHandler.RoundTripper(transport, r.cfg.RegistryType)
}

// ListTags lists all tags for a given repository, returning a list of tag names.
// This uses google/go-containerregistry which properly handles pagination and
// bearer token authentication. The transport is built from the same configuration
// as the docker-registry-client (timeouts, TLS, metrics), but uses
// go-containerregistry's native authentication instead of the docker-registry-client
// auth wrappers.
//
// This function does not impose an overall timeout. The transport's per-request
// timeouts (DialTimeout, ResponseHeaderTimeout) protect against hung requests.
// Callers needing an overall timeout should pass a context with deadline.
// ListTags lists tags for a repository using the Docker Registry V2 API directly.
// This replaces go-containerregistry/pkg/v1/remote.List (which pulled in 558 deps)
// with a simple HTTP GET to /v2/<repo>/tags/list.
func (r *Registry) ListTags(ctx context.Context, repository string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	url := fmt.Sprintf("%s/v2/%s/tags/list", r.url, repository)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "creating tags list request for %q", repository)
	}

	username, password := r.cfg.GetCredentials()
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := r.buildTransport().RoundTrip(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list tags for %q", repository)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, errors.Errorf("failed to list tags for %q: %d %s", repository, resp.StatusCode, body)
	}

	var tagList struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tagList); err != nil {
		return nil, errors.Wrapf(err, "decoding tags for %q", repository)
	}
	return tagList.Tags, nil
}
