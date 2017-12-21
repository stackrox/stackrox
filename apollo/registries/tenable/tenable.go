package tenable

import (
	"errors"
	"strings"

	"bitbucket.org/stack-rox/apollo/apollo/registries"
	registryTypes "bitbucket.org/stack-rox/apollo/apollo/registries/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/heroku/docker-registry-client/registry"
)

var (
	remote         = "registry.cloud.tenable.com"
	remoteEndpoint = "https://" + remote
)

var (
	log = logging.New("registry/docker")
)

type tenableRegistry struct {
	protoRegistry *v1.Registry

	hub      *registry.Registry
	registry string
}

func newRegistry(protoRegistry *v1.Registry) (*tenableRegistry, error) {
	accessKey, ok := protoRegistry.Config["accessKey"]
	if !ok {
		return nil, errors.New("Config parameter 'accessKey' must be defined for Tenable registries")
	}
	secretKey, ok := protoRegistry.Config["secretKey"]
	if !ok {
		return nil, errors.New("Config parameter 'secretKey' must be defined for Tenable registries")
	}
	tran, err := newPersistentTokenTransport(remoteEndpoint, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	hub, err := registry.NewFromTransport(remoteEndpoint, accessKey, secretKey, tran, registry.Log)
	if err != nil {
		return nil, err
	}
	return &tenableRegistry{
		protoRegistry: protoRegistry,
		hub:           hub,
		registry:      protoRegistry.Endpoint,
	}, nil
}

// Metadata returns the metadata via this registries implementation
func (d *tenableRegistry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	if image == nil {
		return nil, nil
	}
	manifest, err := d.hub.ManifestV2(image.GetRemote(), image.GetTag())
	if err != nil {
		return nil, err
	}
	image.Sha = strings.TrimPrefix(manifest.Config.Digest.String(), "sha256:")
	return nil, nil
}

// ProtoRegistry returns the Proto Registry this registry is based on
func (d *tenableRegistry) ProtoRegistry() *v1.Registry {
	return d.protoRegistry
}

// Test tests the current registry and makes sure that it is working properly
func (d *tenableRegistry) Test() error {
	_, err := d.hub.Repositories()
	return err
}

// Match decides if the image is contained within this registry
func (d *tenableRegistry) Match(image *v1.Image) bool {
	return remote == image.Registry
}

func init() {
	registries.Registry["tenable"] = func(registry *v1.Registry) (registryTypes.ImageRegistry, error) {
		reg, err := newRegistry(registry)
		return reg, err
	}
}
