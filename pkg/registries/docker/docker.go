package docker

import (
	"fmt"
	"strings"
	"time"

	manifestV1 "github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	registryTimeout = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "docker", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := NewDockerRegistry(integration)
		return reg, err
	}
}

// Registry is the basic docker registry implementation
type Registry struct {
	cfg                   Config
	protoImageIntegration *storage.ImageIntegration

	Client *registry.Registry

	url      string
	registry string // This is the registry portion of the image
}

// Config is the basic config for the docker registry
type Config struct {
	// Endpoint defines the Docker Registry URL
	Endpoint string
	// Username defines the Username for the Docker Registry
	Username string
	// Password defines the password for the Docker Registry
	Password string
	// Insecure defines if the registry should be insecure
	Insecure bool
}

// NewDockerRegistryWithConfig creates a new instantiation of the docker registry
// TODO(cgorman) AP-386 - properly put the base docker registry into another pkg
func NewDockerRegistryWithConfig(cfg Config, integration *storage.ImageIntegration) (*Registry, error) {
	endpoint := cfg.Endpoint
	if strings.EqualFold(endpoint, "https://docker.io") || strings.EqualFold(endpoint, "docker.io") {
		endpoint = "https://registry-1.docker.io"
	}
	url, err := urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	// if the registryServer endpoint contains docker.io then the image will be docker.io/namespace/repo:tag
	registryServer := urlfmt.GetServerFromURL(url)
	if strings.Contains(cfg.Endpoint, "docker.io") {
		registryServer = "docker.io"
	}
	var client *registry.Registry
	if cfg.Insecure {
		client, err = registry.NewInsecure(url, cfg.Username, cfg.Password)
	} else {
		client, err = registry.New(url, cfg.Username, cfg.Password)
	}
	if err != nil {
		return nil, err
	}

	// Turn off the logs
	client.Logf = registry.Quiet

	client.Client.Timeout = registryTimeout

	return &Registry{
		url:                   url,
		registry:              registryServer,
		Client:                client,
		cfg:                   cfg,
		protoImageIntegration: integration,
	}, nil
}

// NewDockerRegistry creates a generic docker registry integration
func NewDockerRegistry(integration *storage.ImageIntegration) (*Registry, error) {
	dockerConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Docker)
	if !ok {
		return nil, fmt.Errorf("Docker configuration required")
	}
	cfg := Config{
		Endpoint: dockerConfig.Docker.GetEndpoint(),
		Username: dockerConfig.Docker.GetUsername(),
		Password: dockerConfig.Docker.GetPassword(),
		Insecure: dockerConfig.Docker.GetInsecure(),
	}
	return NewDockerRegistryWithConfig(cfg, integration)
}

// Match decides if the image is contained within this registry
func (r *Registry) Match(image *storage.Image) bool {
	return urlfmt.TrimHTTPPrefixes(r.registry) == image.GetName().GetRegistry()
}

// Global returns whether or not this registry is available from all clusters
func (r *Registry) Global() bool {
	return len(r.protoImageIntegration.GetClusters()) == 0
}

// Metadata returns the metadata via this registries implementation
func (r *Registry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	log.Infof("Getting metadata for image %s", image.GetName().GetFullName())
	if image == nil {
		return nil, nil
	}

	remote := image.GetName().GetRemote()
	digest, manifestType, err := r.Client.ManifestDigest(remote, utils.Reference(image))
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get the manifest digest ")
	}

	switch manifestType {
	case manifestV1.MediaTypeManifest:
		return r.HandleV1Manifest(remote, digest.String())
	case manifestV1.MediaTypeSignedManifest:
		return r.HandleV1SignedManifest(remote, digest.String())
	case registry.MediaTypeManifestList:
		return r.HandleV2ManifestList(remote, digest.String())
	case schema2.MediaTypeManifest:
		return r.HandleV2Manifest(remote, digest.String())
	default:
		return nil, fmt.Errorf("unknown manifest type '%s'", manifestType)
	}
}

// Test tests the current registry and makes sure that it is working properly
func (r *Registry) Test() error {
	return r.Client.Ping()
}

// Config returns the configuration of the docker registry
func (r *Registry) Config() *types.Config {
	return &types.Config{
		Username:         r.cfg.Username,
		Password:         r.cfg.Password,
		Insecure:         r.cfg.Insecure,
		URL:              r.url,
		RegistryHostname: r.registry,
	}
}
