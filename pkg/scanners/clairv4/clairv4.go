package clairv4

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	requestTimeout = 30 * time.Second
	typeString     = "clairV4"

	indexStatePath          = "/indexer/api/v1/index_state"
	indexReportPath         = "/indexer/api/v1/index_report"
	indexPath               = "/indexer/api/v1/index_report"
	vulnerabilityReportPath = "/matcher/api/v1/vulnerability_report"
)

var (
	log = logging.LoggerForModule()

	errNoMetadata = errors.New("Unable to complete scan because the image is missing metadata")
	errInternal   = errors.New("Internal error")
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

	testEndpoint                string
	indexReportEndpoint         string
	indexEndpoint               string
	vulnerabilityReportEndpoint string

}

func newScanner(integration *storage.ImageIntegration) (*clairv4, error) {
	cfg := integration.GetClairV4()
	if cfg == nil {
		return nil, errors.New("Clair v4 configuration required")
	}
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
		},
	}

	scanner := &clairv4{
		client:                client,
		protoImageIntegration: integration,

		testEndpoint:                path.Join(endpoint, indexStatePath),
		indexReportEndpoint:         path.Join(endpoint, indexReportPath),
		indexEndpoint:               path.Join(endpoint, indexPath),
		vulnerabilityReportEndpoint: path.Join(endpoint, vulnerabilityReportPath),

		ScanSemaphore: types.NewDefaultSemaphore(),
	}
	return scanner, nil
}

func validate(cfg *storage.ClairV4Config) error {
	errorList := errorhelpers.NewErrorList("Clair v4 Validation")
	if cfg.GetEndpoint() == "" {
		errorList.AddString("Endpoint must be specified")
	}
	return errorList.ToError()
}

func (c *clairv4) GetScan(image *storage.Image) (*storage.ImageScan, error) {
	if image.GetMetadata() == nil {
		return nil, errNoMetadata
	}

	// For logging/error message purposes.
	imgName := image.GetName().GetFullName()

	digest, err := claircore.ParseDigest(imageutils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrapf(err, "parsing image digest for image %s", imgName)
	}

	exists, err := c.indexReportExists(digest)
	// Exit early if this is an unexpected status code error.
	// Otherwise, continue as if the index report does not already exist.
	if isUnexpectedStatusCodeError(err) {
		return nil, errors.Wrapf(err, "checking if index report exists for Clair v4 scan of %s", imgName)
	}

	if exists {
		manifest, err := manifestForImage(image)
		if err != nil {
			return nil, errors.Wrapf(err, "creating manifest for Clair v4 scan of %s", imgName)
		}

		if err := c.index(manifest); err != nil {
			return nil, errors.Wrapf(err, "indexing manifest for Clair v4 scan of %s", imgName)
		}
	}

	report, err := c.getVulnerabilityReport(digest)
	if err != nil {
		return nil, errors.Wrapf(err, "getting vulnerability report for Clair v4 scan of %s", imgName)
	}

	return imageScanFromReport(report), nil
}

func (c *clairv4) indexReportExists(digest claircore.Digest) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, path.Join(c.indexReportEndpoint, digest.String()), nil)
	if err != nil {
		return false, err
	}

	var exists bool
	// Ignore any error returned. Just assume
	err = retry.WithRetry(func() error {
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer utils.IgnoreError(resp.Body.Close)

		switch resp.StatusCode {
		case http.StatusNotModified:
			exists = true
			return nil
		case http.StatusOK, http.StatusNotFound:
			return nil
		case http.StatusInternalServerError:
			return retry.MakeRetryable(errInternal)
		default:
			return newUnexpectedStatusCodeError(resp.StatusCode)
		}
	}, retry.Tries(3), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())

	return exists, err
}

func (c *clairv4) index(manifest *claircore.Manifest) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.indexEndpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}

	return retry.WithRetry(func() error {
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer utils.IgnoreError(resp.Body.Close)

		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated:
			// The index report was created, hopefully...
		case http.StatusInternalServerError:
			return retry.MakeRetryable(errInternal)
		default:
			return newUnexpectedStatusCodeError(resp.StatusCode)
		}

		var ir claircore.IndexReport
		if err := json.NewDecoder(resp.Body).Decode(&ir); err != nil {
			return err
		}
		if !ir.Success && ir.Err != "" {
			return errors.New(ir.Err)
		}

		return nil
	}, retry.Tries(3), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())
}

func (c *clairv4) getVulnerabilityReport(digest claircore.Digest) (*claircore.VulnerabilityReport, error) {
	req, err := http.NewRequest(http.MethodGet, path.Join(c.vulnerabilityReportEndpoint, digest.String()), nil)
	if err != nil {
		return nil, err
	}

	var vulnReport claircore.VulnerabilityReport
	// Ignore any error returned. Just assume
	err = retry.WithRetry(func() error {
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer utils.IgnoreError(resp.Body.Close)

		switch resp.StatusCode {
		case http.StatusOK:
		case http.StatusAccepted, http.StatusInternalServerError:
			// http.StatusAccepted is treated like an internal error.
			return retry.MakeRetryable(errInternal)
		default:
			return newUnexpectedStatusCodeError(resp.StatusCode)
		}

		return json.NewDecoder(resp.Body).Decode(&vulnReport)
	}, retry.Tries(3), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())

	return &vulnReport, err
}

func (c *clairv4) Match(_ *storage.ImageName) bool {
	return true
}

func (c *clairv4) Test() error {
	req, err := http.NewRequest(http.MethodGet, c.testEndpoint, nil)
	if err != nil {
		return fmt.Errorf("unable to create test request: %w", err)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("test request could not be completed: %w", err)
	}
	defer utils.IgnoreError(resp.Body.Close)

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNotModified:
		return nil
	default:
		return fmt.Errorf("received status code %v", resp.StatusCode)
	}
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
