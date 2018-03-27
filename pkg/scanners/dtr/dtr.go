package dtr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"bitbucket.org/stack-rox/apollo/pkg/scanners"
	"bitbucket.org/stack-rox/apollo/pkg/urlfmt"
)

const (
	metadataRefreshInterval = 5 * time.Minute
	requestTimeout          = 5 * time.Second
)

var (
	log = logging.LoggerForModule()
)

type dtr struct {
	client         *http.Client
	metadataTicker *time.Ticker

	server   string
	username string
	password string
	registry string

	protoImageIntegration *v1.ImageIntegration

	metadata *scannerMetadata
	features *metadataFeatures
}

func newScanner(protoImageIntegration *v1.ImageIntegration) (*dtr, error) {
	username, ok := protoImageIntegration.Config["username"]
	if !ok {
		return nil, errors.New("username parameter must be defined for DTR")
	}
	password, ok := protoImageIntegration.Config["password"]
	if !ok {
		return nil, errors.New("password parameter must be defined for DTR")
	}
	endpoint, ok := protoImageIntegration.Config["endpoint"]
	if !ok {
		return nil, errors.New("endpoint parameter must be defined for DTR")
	}

	client := &http.Client{
		Timeout: requestTimeout,
	}
	// Trim any trailing slashes as the expectation will be that the input is in the form
	// https://12.12.12.12:8080 or https://dtr.com
	endpoint, err := urlfmt.FormatURL(endpoint, true, false)
	if err != nil {
		return nil, err
	}
	registry := urlfmt.GetServerFromURL(endpoint)
	scanner := &dtr{
		client:                client,
		server:                endpoint,
		username:              username,
		registry:              registry,
		password:              password,
		metadataTicker:        time.NewTicker(metadataRefreshInterval),
		protoImageIntegration: protoImageIntegration,
	}

	if err := scanner.fetchMetadata(); err != nil {
		return nil, err
	}

	go scanner.refreshMetadata()
	return scanner, nil
}

func parseMetadata(body []byte) (*scannerMetadata, error) {
	var meta scannerMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func parseFeatures(body []byte) (*metadataFeatures, error) {
	var meta metadataFeatures
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (d *dtr) refreshMetadata() {
	for range d.metadataTicker.C {
		if err := d.fetchMetadata(); err != nil {
			log.Error(err)
		}
	}
}

func (d *dtr) fetchMetadata() error {
	meta, features, err := d.getStatus()
	if err != nil {
		return err
	}
	d.metadata = meta
	d.features = features
	return nil
}

func (d *dtr) sendRequest(method, urlPrefix string) ([]byte, error) {
	req, err := http.NewRequest(method, d.server+urlPrefix, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(d.username, d.password)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return body, nil
}

func (d *dtr) getStatus() (*scannerMetadata, *metadataFeatures, error) {
	body, err := d.sendRequest("GET", "/api/v0/imagescan/status")
	if err != nil {
		return nil, nil, err
	}
	meta, err := parseMetadata(body)
	if err != nil {
		return nil, nil, err
	}
	body, err = d.sendRequest("GET", "/api/v0/meta/features")
	features, err := parseFeatures(body)
	if err != nil {
		return nil, nil, err
	}
	return meta, features, nil
}

// GetScan takes in an id and returns the image scan for that id if applicable
func (d *dtr) GetScans(image *v1.Image) ([]*v1.ImageScan, error) {
	if image == nil || image.GetName().GetRemote() == "" || image.GetName().GetTag() == "" {
		return nil, nil
	}
	getScanURL := fmt.Sprintf("/api/v0/imagescan/repositories/%v/%v?detailed=true", image.GetName().GetRemote(), image.GetName().GetTag())
	body, err := d.sendRequest("GET", getScanURL)
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
		return nil, fmt.Errorf("expected to receive at least one scan for %v", image.String())
	}
	// After should sort in descending order based on completion
	sort.SliceStable(scans, func(i, j int) bool { return scans[i].CheckCompletedAt.After(scans[j].CheckCompletedAt) })
	return convertTagScanSummariesToImageScans(d.server, scans), nil
}

//GET /api/v0/imagescan/repositories/{namespace}/{reponame}/{tag}?detailed=true
// Scan initiates a scan of the passed id
func (d *dtr) Scan(image *v1.Image) error {
	_, err := d.sendRequest("POST", fmt.Sprintf("/api/v0/imagescan/scan/%v/%v/linux/amd64", image.GetName().GetRemote(), image.GetName().GetTag()))
	if err != nil {
		return err
	}
	return nil
}

// Test initiates a test of the DTR which verifies that we have the proper scan permissions
func (d *dtr) Test() error {
	_, features, err := d.getStatus()
	if err != nil {
		return err
	}
	if !features.ScanningEnabled {
		return errors.New("Scanning is not currently enabled on your Docker Trusted Registry")
	}
	return nil
}

// GetLastScan retrieves the most recent scan
func (d *dtr) GetLastScan(image *v1.Image) (*v1.ImageScan, error) {
	log.Infof("Getting latest scan for image %v", image)
	imageScans, err := d.GetScans(image)
	if err != nil {
		return nil, err
	}
	if len(imageScans) == 0 {
		return nil, fmt.Errorf("no scans were found for image %v", image.String())
	}
	return imageScans[0], nil
}

// Match decides if the image is contained within this registry
func (d *dtr) Match(image *v1.Image) bool {
	return d.registry == image.GetName().GetRegistry()
}

func (d *dtr) Global() bool {
	return len(d.protoImageIntegration.GetClusters()) == 0
}

func init() {
	scanners.Registry["dtr"] = func(integration *v1.ImageIntegration) (scanners.ImageScanner, error) {
		scan, err := newScanner(integration)
		return scan, err
	}
}
