package tenable

import (
	"fmt"

	manifestV2 "github.com/docker/distribution/manifest/schema2"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/transports"
)

var (
	remote         = "registry.cloud.tenable.com"
	remoteEndpoint = "https://" + remote
)

// Creator provides the type and registries.Creator to add to the registries Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageRegistry, error)) {
	return "tenable", func(integration *storage.ImageIntegration) (types.ImageRegistry, error) {
		reg, err := newRegistry(integration)
		return reg, err
	}
}

type tenableRegistry struct {
	protoImageIntegration *storage.ImageIntegration

	client *registry.Registry

	config    *storage.TenableConfig
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

func validate(config *storage.TenableConfig) error {
	errorList := errorhelpers.NewErrorList("Tenable Validation")
	if config.GetAccessKey() == "" {
		errorList.AddString("Access key must be specified for Tenable scanner")
	}
	if config.GetSecretKey() == "" {
		errorList.AddString("Secret Key must be specified for Tenable scanner")
	}
	return errorList.ToError()
}

func newRegistry(integration *storage.ImageIntegration) (*tenableRegistry, error) {
	tenableConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Tenable)
	if !ok {
		return nil, fmt.Errorf("Tenable configuration required")
	}
	config := tenableConfig.Tenable
	if err := validate(config); err != nil {
		return nil, err
	}
	tran, err := transports.NewPersistentTokenTransport(remoteEndpoint, config.GetAccessKey(), config.GetSecretKey())
	if err != nil {
		return nil, err
	}

	reg, err := registry.NewFromTransport(remoteEndpoint, tran, registry.Log)
	if err != nil {
		return nil, err
	}

	return &tenableRegistry{
		config:                config,
		client:                reg,
		transport:             tran,
		protoImageIntegration: integration,
	}, nil
}

// Metadata returns the metadata via this registries implementation
func (d *tenableRegistry) Metadata(image *storage.Image) (*storage.ImageMetadata, error) {
	if image == nil {
		return nil, nil
	}
	manifest, err := d.client.ManifestV2(image.GetName().GetRemote(), utils.Reference(image))
	if err != nil {
		return nil, err
	}
	digest := imageTypes.NewDigest(manifest.Config.Digest.String()).Digest()
	layers := make([]string, 0, len(manifest.Layers))
	for _, layer := range manifest.Layers {
		layers = append(layers, layer.Digest.String())
	}
	return &storage.ImageMetadata{
		V2: &storage.V2Metadata{
			Digest: digest,
		},
		LayerShas: layers,
	}, nil
}

// Test tests the current registry and makes sure that it is working properly
func (d *tenableRegistry) Test() error {
	_, err := d.client.Repositories()
	return err
}

// Match decides if the image is contained within this registry
func (d *tenableRegistry) Match(image *storage.Image) bool {
	return remote == image.GetName().GetRegistry()
}

func (d *tenableRegistry) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func (d *tenableRegistry) Config() *types.Config {
	// Tenable cannot be used to pull down scans
	return nil
}
