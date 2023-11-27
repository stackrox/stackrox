package clairv4

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/quay/claircore"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	imageutils "github.com/stackrox/rox/pkg/images/utils"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries"
	"github.com/stackrox/rox/pkg/retry"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	requestTimeout = 2 * time.Minute
	typeString     = "clairV4"

	indexStatePath          = "/indexer/api/v1/index_state"
	indexReportPath         = "/indexer/api/v1/index_report"
	indexPath               = "/indexer/api/v1/index_report"
	vulnerabilityReportPath = "/matcher/api/v1/vulnerability_report"

	httpRequestRetryCount = 3
)

var (
	log = logging.LoggerForModule()

	errInternal = errors.New("Clair v4: Clair internal server error")
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
	// TODO: Probably something along these lines is required here? // MC
	//
	// func getTLSConfig() (*tls.Config, error) {
	// 	tlsConfig, err := clientconn.TLSConfig(mtls.ScannerSubject, clientconn.TLSConfigOptions{
	// 		UseClientCert: clientconn.MustUseClientCert,
	// 	})
	// 	if err != nil {
	// 		return nil, errors.Wrap(err, "failed to initialize TLS config")
	// 	}
	// 	return tlsConfig, nil
	// }

	endpoint := urlfmt.FormatURL(cfg.GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	client := &http.Client{
		// No need to specify a context for HTTP requests, as the client specifies a request timeout.
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: cfg.GetInsecure(),
			},
			Proxy: proxy.FromConfig(),
			// The following values are taken from http.DefaultTransport as of go1.19.3.
			ForceAttemptHTTP2:     true,
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
	// For logging/error message purposes.
	imgName := image.GetName().GetFullName()

	if image.GetMetadata() == nil {
		return nil, errors.Errorf("Clair v4: Unable to complete scan of image %s because it is missing metadata", imgName)
	}

	// Use claircore.ParseDigest instead of types.Digest (see pkg/images/types/digest.go)
	// to mirror clairctl (https://github.com/quay/clair/blob/v4.5.0/cmd/clairctl/report.go#L251).
	ccDigest, err := claircore.ParseDigest(imageutils.GetSHA(image))
	if err != nil {
		return nil, errors.Wrapf(err, "Clair v4: parsing image digest for image %s", imgName)
	}
	digest := ccDigest.String()

	exists, err := c.indexReportExists(digest)
	if err != nil {
		log.Debugf("Clair v4: Received error status from Clair: %v", err)
	}
	// Exit early if this is an unexpected status code error.
	// If it's not an unexpected error, then continue as normal and ignore the error.
	if isUnexpectedStatusCodeError(err) {
		return nil, errors.Wrapf(err, "Clair v4: checking if index report exists for %s", imgName)
	}
	if !exists {
		registry := c.activeRegistries.GetRegistryByImage(image)
		if registry == nil {
			return nil, errors.Errorf("Clair v4: unable to find required registry for %s", imgName)
		}

		// The index report does not exist, so we need to index the image's manifest.
		manifest, err := manifest(registry, image)
		if err != nil {
			return nil, errors.Wrapf(err, "Clair v4: creating manifest for %s", imgName)
		}

		log.Debugf("Manifest for %s: %+v", imgName, manifest)

		if err := c.index(manifest); err != nil {
			return nil, errors.Wrapf(err, "Clair v4: indexing manifest for %s", imgName)
		}
	}

	// Clair v4 should have the image's manifest indexed by now, so get the vulnerability report.
	report, err := c.getVulnerabilityReport(digest)
	if err != nil {
		return nil, errors.Wrapf(err, "Clair v4: getting vulnerability report for %s", imgName)
	}

	return imageScan(report), nil
}

func (c *clairv4) indexReportExists(digest string) (bool, error) {
	// FIXME: go1.19 adds https://pkg.go.dev/net/url#JoinPath, which seems more idiomatic.
	url := strings.Join([]string{c.indexReportEndpoint, digest}, "/")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}

	var exists bool
	err = retry.WithRetry(func() error {
		resp, err := c.client.Do(req)
		if err != nil {
			return err
		}
		defer utils.IgnoreError(resp.Body.Close)

		switch resp.StatusCode {
		case http.StatusNotModified:
			// This is the only status code which indicates the index already exists.
			exists = true
			return nil
		case http.StatusOK, http.StatusNotFound:
			return nil
		case http.StatusInternalServerError:
			return retry.MakeRetryable(errInternal)
		default:
			return newUnexpectedStatusCodeError(resp.StatusCode)
		}
	}, retry.Tries(httpRequestRetryCount), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())

	return exists, err
}

func (c *clairv4) index(manifest *claircore.Manifest) error {
	body, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	return retry.WithRetry(func() error {
		// Make a new request per retry to ensure the body is fully populated for each retry.
		req, err := http.NewRequest(http.MethodPost, c.indexEndpoint, bytes.NewReader(body))
		if err != nil {
			return err
		}

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
			return errors.Errorf("indexing error: %s", ir.Err)
		}

		return nil
	}, retry.Tries(httpRequestRetryCount), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())
}

func (c *clairv4) getVulnerabilityReport(digest string) (*claircore.VulnerabilityReport, error) {
	// FIXME: go1.19 adds https://pkg.go.dev/net/url#JoinPath, which seems more idiomatic.
	url := strings.Join([]string{c.vulnerabilityReportEndpoint, digest}, "/")
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var vulnReport *claircore.VulnerabilityReport
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

		vulnReport = &claircore.VulnerabilityReport{}
		if err := json.NewDecoder(resp.Body).Decode(vulnReport); err != nil {
			return err
		}

		return nil
	}, retry.Tries(httpRequestRetryCount), retry.WithExponentialBackoff(), retry.OnlyRetryableErrors())

	return vulnReport, err
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
	return c.name
}

func (c *clairv4) GetVulnDefinitionsInfo() (*v1.VulnDefinitionsInfo, error) {
	return nil, nil
}
