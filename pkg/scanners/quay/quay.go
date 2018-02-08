package quay

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/images"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
	dockerRegistry "github.com/heroku/docker-registry-client/registry"
)

const (
	requestTimeout = 5 * time.Second
	username       = "$oauthtoken"
)

var (
	log = logging.New("quay")
)

type quay struct {
	client *http.Client

	endpoint   string
	oauthToken string

	reg *dockerRegistry.Registry

	protoScanner *v1.Scanner
}

func newScanner(protoScanner *v1.Scanner) (*quay, error) {
	oauthToken, ok := protoScanner.Config["oauthToken"]
	if !ok {
		return nil, errors.New("'oauthToken' parameter must be defined for Quay.io")
	}
	endpoint, err := urlfmt.FormatURL(protoScanner.GetEndpoint(), true, false)
	if err != nil {
		return nil, err
	}
	reg, err := dockerRegistry.New(endpoint, username, oauthToken)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Timeout: requestTimeout,
	}
	scanner := &quay{
		client:     client,
		reg:        reg,
		endpoint:   endpoint,
		oauthToken: oauthToken,

		protoScanner: protoScanner,
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
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", q.oauthToken))
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
	_, err := q.reg.Repositories()
	return err
}

func (q *quay) ProtoScanner() *v1.Scanner {
	return q.protoScanner
}

func (q *quay) populateSHA(image *v1.Image) error {
	manifest, err := q.reg.ManifestV2(image.GetRemote(), image.GetTag())
	if err != nil {
		return err
	}
	image.Sha = manifest.Config.Digest.String()
	return nil
}

// GetLastScan retrieves the most recent scan
func (q *quay) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	if image == nil || image.GetRemote() == "" || image.GetTag() == "" {
		return nil, nil
	}
	// If SHA is empty, then retrieve it from the Quay registry
	if image.GetSha() == "" {
		if err := q.populateSHA(image); err != nil {
			return nil, fmt.Errorf("unable to retrieve SHA for image %v due to: %+v", images.Wrapper{Image: image}.String(), err)
		}
	}

	values := url.Values{}
	values.Add("features", "true")
	values.Add("vulnerabilities", "true")
	body, status, err := q.sendRequest("GET", values, "api", "v1", "repository", image.GetRemote(), "manifest", images.Wrapper{Image: image}.GetPrefixedSHA(), "security")
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("Unexpected status code %d when retrieving image scan for %s", status, images.Wrapper{Image: image})
	}
	scan, err := parseImageScan(body)
	if err != nil {
		return nil, err
	}
	if scan.Data.Layer == nil {
		return nil, fmt.Errorf("Layer for image %s was not found", images.Wrapper{Image: image})
	}
	return convertScanToImageScan(image, scan), nil
}

// Match decides if the image is contained within this scanner
func (q *quay) Match(image *v1.Image) bool {
	return q.protoScanner.GetRemote() == image.GetRegistry()
}

func (q *quay) Global() bool {
	return len(q.protoScanner.GetClusters()) == 0
}

func init() {
	scanners.Registry["quay"] = func(scanner *v1.Scanner) (scanners.ImageScanner, error) {
		scan, err := newScanner(scanner)
		return scan, err
	}
}
