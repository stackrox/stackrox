package tenable

import (
	"errors"
	"sync"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/registries"
	"bitbucket.org/stack-rox/apollo/pkg/transports"
	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
)

var (
	remote         = "registry.cloud.tenable.com"
	remoteEndpoint = "https://" + remote
)

var (
	log = logging.LoggerForModule()
)

type tenableRegistry struct {
	protoImageIntegration *v1.ImageIntegration

	getClientOnce sync.Once
	clientObj     client

	accessKey string
	secretKey string
	transport *transports.PersistentTokenTransport
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

func newRegistry(integration *v1.ImageIntegration) (*tenableRegistry, error) {
	accessKey, ok := integration.Config["accessKey"]
	if !ok {
		return nil, errors.New("Config parameter 'accessKey' must be defined for Tenable registries")
	}
	secretKey, ok := integration.Config["secretKey"]
	if !ok {
		return nil, errors.New("Config parameter 'secretKey' must be defined for Tenable registries")
	}
	tran, err := transports.NewPersistentTokenTransport(remoteEndpoint, accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	return &tenableRegistry{
		accessKey:             accessKey,
		secretKey:             secretKey,
		transport:             tran,
		protoImageIntegration: integration,
	}, nil
}

func (d *tenableRegistry) client() client {
	d.getClientOnce.Do(func() {
		reg, err := registry.NewFromTransport(remoteEndpoint, d.transport, registry.Log)
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
	manifest, err := d.client().ManifestV2(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		return nil, err
	}
	digest := images.NewDigest(manifest.Config.Digest.String()).Digest()
	layers := make([]string, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layers = append(layers, layer.Digest.String())
	}
	return &v1.ImageMetadata{
		RegistrySha: digest,
		V2: &v1.V2Metadata{
			Digest: digest,
			Layers: layers,
		},
	}, nil
}

// Test tests the current registry and makes sure that it is working properly
func (d *tenableRegistry) Test() error {
	_, err := d.client().Repositories()
	return err
}

// Match decides if the image is contained within this registry
func (d *tenableRegistry) Match(image *v1.Image) bool {
	return remote == image.GetName().GetRegistry()
}

func (d *tenableRegistry) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func (d *tenableRegistry) Config() *registries.Config {
	// Tenable cannot be used to pull down scans
	return nil
}

func init() {
	registries.Registry["tenable"] = func(integration *v1.ImageIntegration) (registries.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}
