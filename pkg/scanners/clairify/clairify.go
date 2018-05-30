package clairify

import (
	"fmt"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	clairConv "bitbucket.org/stack-rox/apollo/pkg/clair"
	"bitbucket.org/stack-rox/apollo/pkg/errorhelpers"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/resolver"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	"bitbucket.org/stack-rox/clairify/client"
	"bitbucket.org/stack-rox/clairify/types"
	clairV1 "github.com/coreos/clair/api/v1"
)

var (
	logger = logging.LoggerForModule()
)

func validateConfig(c *v1.ClairifyConfig) error {
	var errors []string
	if c.GetEndpoint() == "" {
		errors = append(errors, "endpoint parameter must be defined for Clairify")
	}
	// This is purposefully done because the registry is optional
	if reg := c.GetRegistry(); reg != nil {
		if reg.GetImageRepo() == "" {
			errors = append(errors, "image repo parameter must be defined for Clairify registry")
		}
		if reg.GetUrl() == "" {
			errors = append(errors, "url parameter must be defined for Clairify registry")
		}
	}
	return errorhelpers.FormatErrorStrings("Validation", errors)
}

type clairify struct {
	client                *client.Clairify
	conf                  *v1.ClairifyConfig
	protoImageIntegration *v1.ImageIntegration
}

func newScanner(protoImageIntegration *v1.ImageIntegration) (*clairify, error) {
	clairifyConfig, ok := protoImageIntegration.IntegrationConfig.(*v1.ImageIntegration_Clairify)
	if !ok {
		return nil, fmt.Errorf("Clairify configuration required")
	}
	conf := clairifyConfig.Clairify
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	endpoint, err := urlfmt.FormatURL(conf.Endpoint, false, false)
	if err != nil {
		return nil, err
	}

	client := client.New(endpoint, true)
	if err := client.Ping(); err != nil {
		return nil, err
	}
	scanner := &clairify{
		client: client,
		conf:   conf,
		protoImageIntegration: protoImageIntegration,
	}
	return scanner, nil
}

// Test initiates a test of the Clairify Scanner which verifies that we have the proper scan permissions
func (c *clairify) Test() error {
	return c.client.Ping()
}

func convertLayerToImageScan(layerEnvelope *clairV1.LayerEnvelope) *v1.ImageScan {
	return &v1.ImageScan{
		Components: clairConv.ConvertFeatures(layerEnvelope.Layer.Features),
	}
}

// Image contains image naming metadata.
type Image struct {
	SHA       string `json:"sha"`
	Registry  string `json:"registry"`
	Namespace string `json:"namespace"`
	Repo      string `json:"repo"`
	Tag       string `json:"tag"`
}

func v1ImageToClairifyImage(i *v1.Image) *types.Image {
	return &types.Image{
		SHA:       i.GetMetadata().GetRegistrySha(),
		Registry:  i.GetName().GetRegistry(),
		Namespace: images.Wrapper{Image: i}.Namespace(),
		Repo:      images.Wrapper{Image: i}.Repo(),
		Tag:       i.GetName().GetTag(),
	}
}

func (c *clairify) getScanBySHA(sha string) (*clairV1.LayerEnvelope, error) {
	return c.client.RetrieveImageDataBySHA(sha, true, true)
}

// Try many ways to retrieve a sha
func (c *clairify) getScan(image *v1.Image) (*clairV1.LayerEnvelope, error) {
	switch {
	case image.GetMetadata().GetRegistrySha() != "":
		if env, err := c.getScanBySHA(image.GetMetadata().GetRegistrySha()); err == nil {
			return env, nil
		}
		fallthrough
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
	if image.GetName().GetRegistry() == c.conf.Registry.GetImageRepo() {
		reg := c.conf.Registry
		_, err := c.client.AddImage(reg.GetUsername(), reg.GetPassword(), &types.ImageRequest{
			Image:    image.GetName().GetFullName(),
			Registry: resolver.Registry(reg.GetUrl(), reg.GetInsecure()),
		})
		return err
	}
	_, err := c.client.AddImage("", "", &types.ImageRequest{
		Image:    image.GetName().GetFullName(),
		Registry: resolver.Registry(image.GetName().GetRegistry(), false),
	})
	return err
}

// Match decides if the image is contained within this scanner
func (c *clairify) Match(image *v1.Image) bool {
	return true
}

func (c *clairify) Global() bool {
	return len(c.protoImageIntegration.GetClusters()) == 0
}

func init() {
	scanners.Registry["clairify"] = func(integration *v1.ImageIntegration) (scanners.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}
