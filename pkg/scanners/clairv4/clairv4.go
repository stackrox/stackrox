package clairv4

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/pkg/errors"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	// TODO: determine a good timeout.
	requestTimeout = 10 * time.Second
	typeString     = "clairv4"

	indexStatePath = "index_state"
)

var (
	errNoMetadata = errors.New("Unable to complete scan because the image is missing metadata")
)

// Creator provides the type a scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

var _ types.Scanner = (*clairv4)(nil)

type clairv4 struct {
	types.ScanSemaphore

	client                *http.Client
	protoImageIntegration *storage.ImageIntegration

	testEndpoint string
}

func newScanner(integration *storage.ImageIntegration) (*clairv4, error) {
	config := integration.GetClairV4()
	if config == nil {
		return nil, errors.New("Clair v4 configuration required")
	}
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

	scanner := &clairv4{
		client:                client,
		protoImageIntegration: integration,

		testEndpoint: path.Join(endpoint, indexStatePath),

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func validate(clairv4 *storage.ClairV4Config) error {
	errorList := errorhelpers.NewErrorList("Clair v4 Validation")
	if clairv4.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified")
	}
	return errorList.ToError()
}

func (c *clairv4) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image.GetMetadata() == nil {
		return nil, errNoMetadata
	}

	return nil, nil
}

func (c *clairv4) Match(_ *storage.ImageName) bool {
	return true
}

func (c *clairv4) Test() error {
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.testEndpoint, nil)
	if err != nil {
		return fmt.Errorf("unable to create test request: %v", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("test request could not be completed: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received status code %v, but expected 200", resp.StatusCode)
	}
	return nil
}

func (c *clairv4) Type() string {
	return typeString
}

func (c *clairv4) Name() string {
	return c.protoImageIntegration.GetName()
}

func (c *clairv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
