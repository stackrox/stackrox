package clair

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	clairV1 "github.com/stackrox/scanner/api/v1"
)

const (
	requestTimeout = 10 * time.Second
	typeString     = "clair"
)

var (
	errNotExists = errors.New("Layer does not exist")
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type clair struct {
	types.ScanSemaphore

	client                *http.Client
	endpoint              string
	protoImageIntegration *storage.ImageIntegration
}

func validate(clair *storage.ClairConfig) error {
	errorList := errorhelpers.NewErrorList("Clair Validation")
	if clair.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified")
	}
	return errorList.ToError()
}

func newScanner(integration *storage.ImageIntegration) (*clair, error) {
	clairConfig, ok := integration.IntegrationConfig.(*storage.ImageIntegration_Clair)
	if !ok {
		return nil, errors.New("Clair configuration required")
	}
	config := clairConfig.Clair
	if err := validate(config); err != nil {
		return nil, err
	}

	endpoint := urlfmt.FormatURL(config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)

	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.GetInsecure(),
			},
			Proxy: proxy.FromConfig(),
		},
	}
	scanner := &clair{
		client:                client,
		endpoint:              endpoint,
		protoImageIntegration: integration,

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func (c *clair) sendRequest(method string, values url.Values, pathSegments ...string) ([]byte, int, error) {
	path, err := urlfmt.FullyQualifiedURL(c.endpoint, values, pathSegments...)
	if err != nil {
		return nil, -1, err
	}
	req, err := http.NewRequest(method, path, nil)
	if err != nil {
		return nil, -1, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, -1, err
	}
	defer utils.IgnoreError(resp.Body.Close)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, errors.Wrap(err, "Error reading Clair response body")
	}
	return body, resp.StatusCode, nil
}

// Test initiates a test of the Clair Scanner which verifies that we have the proper scan permissions
func (c *clair) Test() error {
	_, code, err := c.sendRequest("GET", url.Values{}, "v1", "namespaces")
	if err != nil {
		return err
	} else if code != http.StatusOK {
		return fmt.Errorf("Received status code %v, but expected 200", code)
	}
	return nil
}

func (c *clair) retrieveLayerData(layer string) (*clairV1.LayerEnvelope, error) {
	v := url.Values{}
	v.Add("features", "true")
	v.Add("vulnerabilities", "true")
	body, status, err := c.sendRequest("GET", v, "v1", "layers", layer)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, errNotExists
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code %v: %v", status, string(body))
	}
	le := new(clairV1.LayerEnvelope)
	if err := json.Unmarshal(body, &le); err != nil {
		return nil, err
	}
	return le, nil
}

// GetScan retrieves the most recent scan
func (c *clair) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}
	layers := image.GetMetadata().GetLayerShas()
	if len(layers) == 0 {
		return nil, fmt.Errorf("Cannot get scan for '%s' because no layers were found", image.GetName().GetFullName())
	}
	layerEnvelope, err := c.retrieveLayerData(layers[len(layers)-1])
	if err != nil {
		return nil, err
	}
	return convertLayerToImageScan(image, layerEnvelope), nil
}

// Match decides if the image is contained within this scanner
func (c *clair) Match(_ *storage.ImageName) bool {
	return true
}

func (c *clair) Type() string {
	return typeString
}

func (c *clair) Name() string {
	return c.protoImageIntegration.GetName()
}

func (c *clair) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
