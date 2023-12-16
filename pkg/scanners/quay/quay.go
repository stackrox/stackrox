package quay

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	imageTypes "github.com/stackrox/rox/pkg/images/types"
	imageUtils "github.com/stackrox/rox/pkg/images/utils"
	quayRegistry "github.com/stackrox/rox/pkg/registries/quay"
	registryTypes "github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	requestTimeout = 60 * time.Second
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return types.Quay, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type quay struct {
	client *http.Client

	endpoint   string
	oauthToken string
	registry   registryTypes.Registry

	protoImageIntegration *storage.ImageIntegration
	types.ScanSemaphore
}

func newScanner(protoImageIntegration *storage.ImageIntegration) (*quay, error) {
	quayConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Quay)
	if !ok {
		return nil, errors.New("Quay config must be specified")
	}
	config := quayConfig.Quay

	registry, err := quayRegistry.NewRegistryFromConfig(quayConfig.Quay, protoImageIntegration, false)
	if err != nil {
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
	scanner := &quay{
		client: client,

		registry:   registry,
		endpoint:   endpoint,
		oauthToken: config.GetOauthToken(),

		protoImageIntegration: protoImageIntegration,
		ScanSemaphore:         types.NewDefaultSemaphore(),
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
	defer utils.IgnoreError(resp.Body.Close)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// Test initiates a test of the Quay Scanner which verifies that we have the proper scan permissions
func (q *quay) Test() error {
	return q.registry.Test()
}

// GetScan retrieves the most recent scan
func (q *quay) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}

	values := url.Values{}
	values.Add("features", "true")
	values.Add("vulnerabilities", "true")
	digest := imageTypes.NewDigest(imageUtils.GetSHA(image)).Digest()
	body, status, err := q.sendRequest("GET", values, "api", "v1", "repository", image.GetName().GetRemote(), "manifest", digest, "security")
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d when retrieving image scan for %s", status, imageTypes.Wrapper{GenericImage: image})
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
func (q *quay) Match(image *storage.ImageName) bool {
	return q.registry.Match(image)
}

func (q *quay) Type() string {
	return types.Quay
}

func (q *quay) Name() string {
	return q.protoImageIntegration.GetName()
}

func (q *quay) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
