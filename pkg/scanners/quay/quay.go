package quay

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/stackrox/rox/generated/storage"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/logging"
	quayRegistry "github.com/stackrox/rox/pkg/registries/quay"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	requestTimeout = 5 * time.Second

	typeString = "quay"
)

var log = logging.LoggerForModule()

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageScanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type quay struct {
	client *http.Client

	endpoint   string
	oauthToken string
	registry   registryTypes.ImageRegistry

	protoImageIntegration *storage.ImageIntegration
}

func newScanner(protoImageIntegration *storage.ImageIntegration) (*quay, error) {
	quayConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Quay)
	if !ok {
		return nil, fmt.Errorf("Quay config must be specified")
	}
	config := quayConfig.Quay

	registry, err := quayRegistry.NewRegistryFromConfig(quayConfig.Quay, protoImageIntegration)
	if err != nil {
		return nil, err
	}

	endpoint, err := urlfmt.FormatURL(config.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: requestTimeout,
	}
	scanner := &quay{
		client: client,

		registry:   registry,
		endpoint:   endpoint,
		oauthToken: config.GetOauthToken(),

		protoImageIntegration: protoImageIntegration,
	}
	return scanner, nil
}

func (q *quay) sendRequest(method string, values url.Values, pathSegments ...string) ([]byte, int, error) {
	fullURL, err := urlfmt.FullyQualifiedURL(q.endpoint, values, pathSegments...)
	if err != nil {
		return nil, -1, err
	}
	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, -1, err
	}
	if q.oauthToken != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", q.oauthToken))
	}
	resp, err := q.client.Do(req)
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

// Test initiates a test of the Quay Scanner which verifies that we have the proper scan permissions
func (q *quay) Test() error {
	return q.registry.Test()
}

// GetLastScan retrieves the most recent scan
func (q *quay) GetLastScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}

	values := url.Values{}
	values.Add("features", "true")
	values.Add("vulnerabilities", "true")
	digest := imageTypes.NewDigest(image.GetId()).Digest()
	body, status, err := q.sendRequest("GET", values, "api", "v1", "repository", image.GetName().GetRemote(), "manifest", digest, "security")
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code %d when retrieving image scan for %s", status, imageTypes.Wrapper{Image: image})
	}
	scan, err := parseImageScan(body)
	if err != nil {
		return nil, err
	}
	if scan.Data.Layer == nil {
		return nil, fmt.Errorf("Layer for image %s was not found", image.GetName().GetFullName())
	}
	return convertScanToImageScan(image, scan), nil
}

// Match decides if the image is contained within this scanner
func (q *quay) Match(image *storage.Image) bool {
	return q.registry.Match(image)
}

func (q *quay) Global() bool {
	return len(q.protoImageIntegration.GetClusters()) == 0
}

func (q *quay) Type() string {
	return typeString
}
