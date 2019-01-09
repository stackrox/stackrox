package clairify

import (
	"fmt"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/clairify/client"
	"github.com/stackrox/clairify/types"
	"github.com/stackrox/rox/generated/storage"
	clairConv "github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const typeString = "clairify"

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (scannerTypes.ImageScanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (scannerTypes.ImageScanner, error) {
		scan, err := newScanner(integration, set)
		return scan, err
	}
}

type clairify struct {
	client                *client.Clairify
	conf                  *storage.ClairifyConfig
	protoImageIntegration *storage.ImageIntegration
	activeRegistries      registries.Set
}

func newScanner(protoImageIntegration *storage.ImageIntegration, activeRegistries registries.Set) (*clairify, error) {
	clairifyConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Clairify)
	if !ok {
		return nil, fmt.Errorf("Clairify configuration required")
	}
	conf := clairifyConfig.Clairify
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	endpoint, err := urlfmt.FormatURL(conf.Endpoint, urlfmt.InsecureHTTP, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}

	client := client.New(endpoint, true)
	if err := client.Ping(); err != nil {
		return nil, err
	}
	scanner := &clairify{
		client:                client,
		conf:                  conf,
		protoImageIntegration: protoImageIntegration,
		activeRegistries:      activeRegistries,
	}
	return scanner, nil
}

// Test initiates a test of the Clairify Scanner which verifies that we have the proper scan permissions
func (c *clairify) Test() error {
	return c.client.Ping()
}

func validateConfig(c *storage.ClairifyConfig) error {
	if c.GetEndpoint() == "" {
		return fmt.Errorf("endpoint parameter must be defined for Clairify")
	}
	return nil
}

func convertLayerToImageScan(image *storage.Image, layerEnvelope *clairV1.LayerEnvelope) *storage.ImageScan {
	clairConv.PopulateLayersWithScan(image, layerEnvelope)
	return &storage.ImageScan{
		Components: clairConv.ConvertFeatures(layerEnvelope.Layer.Features),
	}
}

func v1ImageToClairifyImage(i *storage.Image) *types.Image {
	return &types.Image{
		SHA:      i.GetId(),
		Registry: i.GetName().GetRegistry(),
		Remote:   i.GetName().GetRemote(),
		Tag:      i.GetName().GetTag(),
	}
}

func (c *clairify) getScanBySHA(sha string) (*clairV1.LayerEnvelope, error) {
	return c.client.RetrieveImageDataBySHA(sha, true, true)
}

// Try many ways to retrieve a sha
func (c *clairify) getScan(image *storage.Image) (*clairV1.LayerEnvelope, error) {
	if env, err := c.getScanBySHA(image.GetId()); err == nil {
		return env, nil
	}
	switch {
	case image.GetMetadata().GetV2().GetDigest() != "":
		if env, err := c.getScanBySHA(image.GetMetadata().GetV2().GetDigest()); err == nil {
			return env, nil
		}
		fallthrough
	default:
		return c.client.RetrieveImageDataByName(v1ImageToClairifyImage(image), true, true)
	}
}

// GetLastScan retrieves the most recent scan
func (c *clairify) GetLastScan(image *storage.Image) (*storage.ImageScan, error) {
	env, err := c.getScan(image)
	// If not found, then should trigger a scan
	if err != nil {
		if err != client.ErrorScanNotFound {
			return nil, err
		}
		if err := c.scan(image); err != nil {
			return nil, err
		}
		env, err = c.getScan(image)
		if err != nil {
			return nil, err
		}
	}
	return convertLayerToImageScan(image, env), nil
}

func (c *clairify) scan(image *storage.Image) error {
	rc := c.activeRegistries.GetRegistryMetadataByImage(image)
	if rc == nil {
		return nil
	}

	_, err := c.client.AddImage(rc.Username, rc.Password, &types.ImageRequest{
		Image:    image.GetName().GetFullName(),
		Registry: rc.URL,
		Insecure: rc.Insecure})
	return err
}

// Match decides if the image is contained within this scanner
func (c *clairify) Match(image *storage.Image) bool {
	return c.activeRegistries.Match(image)
}

func (c *clairify) Global() bool {
	return len(c.protoImageIntegration.GetClusters()) == 0
}

func (c *clairify) Type() string {
	return typeString
}
