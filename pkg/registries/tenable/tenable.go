package tenable

import (
	"errors"
	"strings"
	"sync"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
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

	registry string

	getClientOnce sync.Once
	clientObj     client

	accessKey string
	secretKey string
	transport *persistentTokenTransport
}

type client interface {
	ManifestV2(repository, reference string) (*manifestV2.DeserializedManifest, error)
	Repositories() ([]string, error)
}

type nilClient struct {
	error error
}

func (n nilClient) ManifestV2(repository, reference string) (*manifestV2.DeserializedManifest, error) {
	return nil, n.error
}

func (n nilClient) Repositories() ([]string, error) {
	return nil, n.error
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

	return &tenableRegistry{
		protoRegistry: protoRegistry,
		registry:      protoRegistry.Endpoint,
		accessKey:     accessKey,
		secretKey:     secretKey,
		transport:     tran,
	}, nil
}

func (d *tenableRegistry) client() client {
	d.getClientOnce.Do(func() {
		reg, err := registry.NewFromTransport(remoteEndpoint, d.accessKey, d.secretKey, d.transport, registry.Log)
		if err != nil {
			d.clientObj = nilClient{err}
			return
		}
		d.clientObj = reg
	})
	return d.clientObj
}

// Metadata returns the metadata via this registries implementation
func (d *tenableRegistry) Metadata(image *v1.Image) (*v1.ImageMetadata, error) {
	if image == nil {
		return nil, nil
	}
	manifest, err := d.client().ManifestV2(image.GetRemote(), image.GetTag())
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
	_, err := d.client().Repositories()
	return err
}

// Match decides if the image is contained within this registry
func (d *tenableRegistry) Match(image *v1.Image) bool {
	return remote == image.Registry
}

func (d *tenableRegistry) Global() bool {
	return len(d.protoRegistry.GetClusters()) == 0
}

func init() {
	registries.Registry["tenable"] = func(registry *v1.Registry) (registries.ImageRegistry, error) {
		reg, err := newRegistry(registry)
		return reg, err
	}
}
