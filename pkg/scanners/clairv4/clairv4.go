package clairv4

import (
	"crypto/tls"
	"net/http"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	requestTimeout = 2 * time.Minute
	typeString     = "clairV4"

	indexStatePath          = "/indexer/api/v1/index_state"
	indexReportPath         = "/indexer/api/v1/index_report"
	indexPath               = "/indexer/api/v1/index_report"
	vulnerabilityReportPath = "/matcher/api/v1/vulnerability_report"
)

// Creator provides the type a scanners.Creator to add to the scanners Registry.
func Creator(set registries.Set) (string, func(integration *storage.ImageIntegration) (types.Scanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.Scanner, error) {
		scan, err := newScanner(integration, set)
		return scan, err
	}
}

var _ types.Scanner = (*clairv4)(nil)

type clairv4 struct {
	types.ScanSemaphore

	name             string
	client           *http.Client
	activeRegistries registries.Set

	testEndpoint                string
	indexReportEndpoint         string
	indexEndpoint               string
	vulnerabilityReportEndpoint string
}

func newScanner(integration *storage.ImageIntegration, activeRegistries registries.Set) (*clairv4, error) {
	cfg := integration.GetClairV4()
	if err := validate(cfg); err != nil {
		return nil, err
	}

	endpoint := urlfmt.FormatURL(cfg.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	client := &http.Client{
		// No need to specify a context for HTTP requests, as the client specifies a request timeout.
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.GetInsecure(),
			},
			Proxy: proxy.FromConfig(),
			// The following values are taken from http.DefaultTransport in go1.19.3.
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	scanner := &clairv4{
		name:             integration.GetName(),
		client:           client,
		activeRegistries: activeRegistries,

		testEndpoint:                endpoint + indexStatePath,
		indexReportEndpoint:         endpoint + indexReportPath,
		indexEndpoint:               endpoint + indexPath,
		vulnerabilityReportEndpoint: endpoint + vulnerabilityReportPath,

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func validate(cfg *storage.ClairV4Config) error {
	errorList := errorhelpers.NewErrorList("Clair v4 Validation")
	if cfg == nil {
		errorList.AddString("configuration required")
	}
	if cfg.GetEndpoint() == "" {
		errorList.AddString("endpoint must be specified")
	}
	return errorList.ToError()
}

func (c *clairv4) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	panic("Unimplemented")
}

func (c *clairv4) Match(_ *storage.ImageName) bool {
	panic("Unimplemented")
}

func (c *clairv4) Test() error {
	panic("Unimplemented")
}

func (c *clairv4) Type() string {
	return typeString
}

func (c *clairv4) Name() string {
	return c.name
}

func (c *clairv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	panic("Unimplemented")
}
