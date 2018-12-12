package clair

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	clairV1 "github.com/coreos/clair/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	requestTimeout = 10 * time.Second
	typeString     = "clair"
)

var (
	log          = logging.LoggerForModule()
	errNotExists = errors.New("Layer does not exist")
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageScanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type clair struct {
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
		return nil, fmt.Errorf("Clair configuration required")
	}
	config := clairConfig.Clair
	if err := validate(config); err != nil {
		return nil, err
	}

	endpoint, err := urlfmt.FormatURL(config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: requestTimeout,
	}
	scanner := &clair{
		client:                client,
		endpoint:              endpoint,
		protoImageIntegration: integration,
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
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
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

// GetLastScan retrieves the most recent scan
func (c *clair) GetLastScan(image *storage.Image) (*storage.ImageScan, error) {
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
func (c *clair) Match(image *storage.Image) bool {
	return true
}

func (c *clair) Global() bool {
	return len(c.protoImageIntegration.GetClusters()) == 0
}

func (c *clair) Type() string {
	return typeString
}
