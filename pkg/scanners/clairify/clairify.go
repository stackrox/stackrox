package clairify

import (
	"fmt"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/clairify/client"
	"github.com/stackrox/clairify/types"
	"github.com/stackrox/rox/generated/api/v1"
	clairConv "github.com/stackrox/rox/pkg/clair"
	"github.com/stackrox/rox/pkg/registries"
	scannerTypes "github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *v1.ImageIntegration) (scannerTypes.ImageScanner, error)) {
	return "clairify", func(integration *v1.ImageIntegration) (scannerTypes.ImageScanner, error) {
		scan, err := newScanner(integration, set)
		return scan, err
	}
}

type clairify struct {
	client                *client.Clairify
	conf                  *v1.ClairifyConfig
	protoImageIntegration *v1.ImageIntegration
	activeRegistries      registries.Set
}

func newScanner(protoImageIntegration *v1.ImageIntegration, activeRegistries registries.Set) (*clairify, error) {
	clairifyConfig, ok := protoImageIntegration.IntegrationConfig.(*v1.ImageIntegration_Clairify)
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

func validateConfig(c *v1.ClairifyConfig) error {
	if c.GetEndpoint() == "" {
		return fmt.Errorf("endpoint parameter must be defined for Clairify")
	}
	return nil
}

func convertLayerToImageScan(layerEnvelope *clairV1.LayerEnvelope) *v1.ImageScan {
	return &v1.ImageScan{
		Components: clairConv.ConvertFeatures(layerEnvelope.Layer.Features),
	}
}

func v1ImageToClairifyImage(i *v1.Image) *types.Image {
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
func (c *clairify) getScan(image *v1.Image) (*clairV1.LayerEnvelope, error) {
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
func (c *clairify) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
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
	return convertLayerToImageScan(env), nil
}

func (c *clairify) scan(image *v1.Image) error {
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
func (c *clairify) Match(image *v1.Image) bool {
	return c.activeRegistries.Match(image)
}

func (c *clairify) Global() bool {
	return len(c.protoImageIntegration.GetClusters()) == 0
}
