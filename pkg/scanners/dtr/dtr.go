package dtr

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/errorhelpers"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/scanners/types"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	requestTimeout = 30 * time.Second
	typeString     = "dtr"
)

var (
	log = logging.LoggerForModule()
)

// Creator provides the type an scanners.Creator to add to the scanners Registry.
func Creator() (string, func(integration *storage.ImageIntegration) (types.ImageScanner, error)) {
	return typeString, func(integration *storage.ImageIntegration) (types.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}

type dtr struct {
	client *http.Client

	conf     config
	registry string

	protoImageIntegration *storage.ImageIntegration
}

type config storage.DTRConfig

func (c config) validate() error {
	errorList := errorhelpers.NewErrorList("Validation")
	if c.Username == "" {
		errorList.AddString("username parameter must be defined for DTR")
	}
	if c.Password == "" {
		errorList.AddString("password parameter must be defined for DTR")
	}
	if c.Endpoint == "" {
		errorList.AddString("endpoint parameter must be defined for DTR")
	}
	return errorList.ToError()
}

func newScanner(protoImageIntegration *storage.ImageIntegration) (*dtr, error) {
	dtrConfig, ok := protoImageIntegration.IntegrationConfig.(*storage.ImageIntegration_Dtr)
	if !ok {
		return nil, fmt.Errorf("DTR configuration required")
	}
	conf := config(*dtrConfig.Dtr)
	if err := conf.validate(); err != nil {
		return nil, err
	}

	// Trim any trailing slashes as the expectation will be that the input is in the form
	// https://12.12.12.12:8080 or https://dtr.com
	var err error
	conf.Endpoint, err = urlfmt.FormatURL(conf.Endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
	if err != nil {
		return nil, err
	}
	registry := urlfmt.GetServerFromURL(conf.Endpoint)
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: conf.Insecure,
			},
		},
	}

	scanner := &dtr{
		client:                client,
		registry:              registry,
		conf:                  conf,
		protoImageIntegration: protoImageIntegration,
	}

	return scanner, nil
}

func (d *dtr) sendRequest(client *http.Client, method, urlPrefix string) ([]byte, error) {
	req, err := http.NewRequest(method, d.conf.Endpoint+urlPrefix, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(d.conf.Username, d.conf.Password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if err := errorFromStatusCode(resp.StatusCode); err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer utils.IgnoreError(resp.Body.Close)
	return body, nil
}

// getScan takes in an id and returns the image scan for that id if applicable
func (d *dtr) getScan(image *storage.Image) (*storage.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}
	getScanURL := fmt.Sprintf("/api/v0/imagescan/repositories/%v/%v?detailed=true", image.GetName().GetRemote(), image.GetName().GetTag())
	body, err := d.sendRequest(d.client, "GET", getScanURL)
	if err != nil {
		return nil, err
	}

	scans, err := parseDTRImageScans(body)
	if err != nil {
		scanErrors, err := parseDTRImageScanErrors(body)
		if err != nil {
			return nil, err
		}
		var errMsg string
		for _, scanErr := range scanErrors.Errors {
			errMsg += scanErr.Message + "\n"
		}
		return nil, errors.New(errMsg)
	}
	if len(scans) == 0 {
		return nil, fmt.Errorf("expected to receive at least one scan for %v", image.GetName().GetFullName())
	}

	// Find the last scan time
	lastScan := scans[0]
	for _, s := range scans {
		if s.CheckCompletedAt.After(lastScan.CheckCompletedAt) {
			lastScan = s
		}
	}
	if lastScan.CheckCompletedAt.IsZero() {
		return nil, fmt.Errorf("expected to receive at least one scan for %s", image.GetName().GetFullName())
	}

	scan := convertTagScanSummaryToImageScan(lastScan)
	// populate V1 Metadata with scan layers
	populateLayersWithScan(image, lastScan.LayerDetails)
	return scan, nil
}

func errorFromStatusCode(status int) error {
	switch status {
	case 400:
		return fmt.Errorf("HTTP 400: Scanning is not enabled")
	case 401:
		return fmt.Errorf("HTTP 401: The client is not authenticated")
	case 405:
		return fmt.Errorf("HTTP 405: Method Not Allowed")
	case 406:
		return fmt.Errorf("HTTP 406: Not Acceptable")
	case 415:
		return fmt.Errorf("HTTP 415: Unsupported Media Type")
	case 200:
	default:
		return nil
	}
	return nil
}

// Test initiates a test of the DTR which verifies that we have the proper scan permissions
func (d *dtr) Test() error {
	client := &http.Client{
		Timeout: requestTimeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: d.conf.Insecure,
			},
		},
	}
	_, err := d.sendRequest(client, "GET", "/api/v0/imagescan/status")
	return err
}

// GetLastScan retrieves the most recent scan
func (d *dtr) GetLastScan(image *storage.Image) (*storage.ImageScan, error) {
	log.Infof("Getting latest scan for image %s", image.GetName().GetFullName())
	return d.getScan(image)
}

// Match decides if the image is contained within this registry
func (d *dtr) Match(image *storage.Image) bool {
	return d.registry == image.GetName().GetRegistry()
}

func (d *dtr) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func (d *dtr) Type() string {
	return typeString
}
