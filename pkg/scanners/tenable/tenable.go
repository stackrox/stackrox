package tenable

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/transports"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
)

const (
	requestTimeout = 5 * time.Second
)

// Variables so we can modify during unit tests
var (
	registry         = "registry.cloud.tenable.com"
	registryEndpoint = "https://" + registry
	apiEndpoint      = "https://cloud.tenable.com"
)

var (
	log = logging.LoggerForModule()
)

type tenable struct {
	client *http.Client

	accessKey string
	secretKey string

	reg *dockerRegistry.Registry

	protoImageIntegration *v1.ImageIntegration
}

func newScanner(protoImageIntegration *v1.ImageIntegration) (*tenable, error) {
	accessKey, ok := protoImageIntegration.Config["accessKey"]
	if !ok {
		return nil, errors.New("'accessKey' parameter must be defined for Tenable.io")
	}
	secretKey, ok := protoImageIntegration.Config["secretKey"]
	if !ok {
		return nil, errors.New("'secretKey' parameter must be defined for Tenable.io")
	}
	tran, err := transports.NewPersistentTokenTransport(registryEndpoint, accessKey, secretKey)
	if err != nil {
		return nil, err
	}
	reg, err := dockerRegistry.NewFromTransport(registryEndpoint, tran, dockerRegistry.Log)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: requestTimeout,
	}
	scanner := &tenable{
		client:    client,
		accessKey: accessKey,
		secretKey: secretKey,
		reg:       reg,
	}
	return scanner, nil
}

func (d *tenable) sendRequest(method, urlPrefix string) ([]byte, int, error) {
	req, err := http.NewRequest(method, apiEndpoint+urlPrefix, nil)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Add("X-ApiKeys", fmt.Sprintf("accessKey=%v; secretKey=%v", d.accessKey, d.secretKey))
	resp, err := d.client.Do(req)
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

// Test initiates a test of the Tenable Scanner which verifies that we have the proper scan permissions
func (d *tenable) Test() error {
	body, status, err := d.sendRequest("GET", "/container-security/api/v1/container/list")
	if err != nil {
		return err
	} else if status != 200 {
		return fmt.Errorf("Unexpected status code '%v' when calling %v. Body: %v",
			status, apiEndpoint+"/container-security/api/v1/container/list", string(body))
	}
	return nil
}

func (d *tenable) populateSHA(image *v1.Image) error {
	manifest, err := d.reg.ManifestV2(image.GetName().GetRemote(), image.GetName().GetTag())
	if err != nil {
		return err
	}
	image.Name.Sha = manifest.Config.Digest.String()
	return nil
}

// GetLastScan retrieves the most recent scan
func (d *tenable) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}
	// If SHA is empty, then retrieve it from the Tenable registry
	if image.GetName().GetSha() == "" {
		if err := d.populateSHA(image); err != nil {
			return nil, fmt.Errorf("unable to retrieve SHA for image %v due to: %+v", images.Wrapper{Image: image}.String(), err)
		}
	}

	getScanURL := fmt.Sprintf("/container-security/api/v1/reports/by_image?image_id=%v",
		images.Wrapper{Image: image}.ShortID())

	body, status, err := d.sendRequest("GET", getScanURL)
	if err != nil {
		return nil, err
	} else if status != 200 {
		return nil, fmt.Errorf("Unexpected status code %v when retrieving image scan: %v", status, string(body))
	}
	scan, err := parseImageScan(body)
	if err != nil {
		return nil, err
	}
	return convertScanToImageScan(image, scan), nil
}

// Match decides if the image is contained within this registry
func (d *tenable) Match(image *v1.Image) bool {
	return registry == image.GetName().GetRegistry()
}

func (d *tenable) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func init() {
	scanners.Registry["tenable"] = func(integration *v1.ImageIntegration) (scanners.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}
