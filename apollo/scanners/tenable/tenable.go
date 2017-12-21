package tenable

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bitbucket.org/stack-rox/apollo/apollo/scanners"
	scannerTypes "bitbucket.org/stack-rox/apollo/apollo/scanners/types"
	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

const (
	requestTimeout = 5 * time.Second
)

// Variables so we can modify during unit tests
var (
	registry    = "registry.cloud.tenable.com"
	apiEndpoint = "https://cloud.tenable.com"
)

var (
	log = logging.New("tenable")
)

type tenable struct {
	client *http.Client

	accessKey string
	secretKey string

	protoScanner *v1.Scanner
}

func newScanner(protoScanner *v1.Scanner) (*tenable, error) {
	accessKey, ok := protoScanner.Config["accessKey"]
	if !ok {
		return nil, errors.New("'accessKey' parameter must be defined for Tenable.io")
	}
	secretKey, ok := protoScanner.Config["secretKey"]
	if !ok {
		return nil, errors.New("'secretKey' parameter must be defined for Tenable.io")
	}
	client := &http.Client{
		Timeout: requestTimeout,
	}
	scanner := &tenable{
		client:    client,
		accessKey: accessKey,
		secretKey: secretKey,

		protoScanner: protoScanner,
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

func (d *tenable) ProtoScanner() *v1.Scanner {
	return d.protoScanner
}

// GetLastScan retrieves the most recent scan
func (d *tenable) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	if image == nil || image.GetRemote() == "" || image.GetTag() == "" {
		return nil, nil
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
	return registry == image.Registry
}

func init() {
	scanners.Registry["tenable"] = func(scanner *v1.Scanner) (scannerTypes.ImageScanner, error) {
		scan, err := newScanner(scanner)
		return scan, err
	}
}
