package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mailru/easyjson"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	v1 "github.com/stackrox/scanner/api/v1"
	protoV1 "github.com/stackrox/scanner/generated/scanner/api/v1"
	"github.com/stackrox/scanner/pkg/clairify/types"
	"github.com/stackrox/scanner/pkg/httputil"
)

// Export these timeouts so that the caller can adjust them as necessary
var (
	GetTimeout  = 20 * time.Second
	ScanTimeout = 2 * time.Minute
	PingTimeout = 5 * time.Second
)

// Clairify is the client for the Clairify extension.
type Clairify struct {
	client   *http.Client
	endpoint string
}

type errorEnvelope struct {
	Error *v1.Error `json:"Error"`
}

// New returns a new Clairify client instance.
func New(endpoint string, insecure bool) *Clairify {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure},
			Proxy:           proxy.TransportFunc,
			// Values are taken from http.DefaultTransport, Go 1.17.3
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	return &Clairify{
		client:   httpClient,
		endpoint: endpoint,
	}
}

// NewWithClient returns a new Clairify client instance based on the passed HTTP client
func NewWithClient(endpoint string, client *http.Client) *Clairify {
	return &Clairify{
		client:   client,
		endpoint: endpoint,
	}
}

func (c *Clairify) sendRequest(request *http.Request, timeout time.Duration) ([]byte, error) {
	request.Header.Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(request.Context(), timeout)
	defer cancel()

	request = request.WithContext(ctx)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrorScanNotFound
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if !httputil.Status2xx(response) {
		return nil, errors.Errorf("Expected status code 2XX. Received %s. Body: %s", response.Status, data)
	}

	var envelope errorEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if envelope.Error != nil {
		return nil, errors.New(envelope.Error.Message)
	}
	return data, nil
}

func encodeValues(opts *types.GetImageDataOpts) url.Values {
	if opts == nil {
		opts = new(types.GetImageDataOpts)
	}

	values := make(url.Values)
	if opts.UncertifiedRHELResults {
		values.Add(types.UncertifiedRHELResultsKey, "true")
	}

	return values
}

// Ping verifies that Clairify is accessible.
func (c *Clairify) Ping() error {
	request, err := http.NewRequest("GET", c.endpoint+"/scanner/ping", nil)
	if err != nil {
		return err
	}
	_, err = c.sendRequest(request, PingTimeout)
	return err
}

// AddImage contacts Clairify to push a specific image to Clair.
func (c *Clairify) AddImage(username, password string, req *types.ImageRequest) (*types.Image, error) {
	// Due to the long timeout for adding an image, always ping before to try to minimize the chance that
	// Clairify is not there
	if err := c.Ping(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequest("POST", c.endpoint+"/scanner/image", bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(username, password)
	imageData, err := c.sendRequest(request, ScanTimeout)
	if err != nil {
		return nil, err
	}

	var imageEnvelope types.ImageEnvelope
	if err := json.Unmarshal(imageData, &imageEnvelope); err != nil {
		return nil, err
	}
	return imageEnvelope.Image, err
}

// RetrieveImageDataBySHA contacts Clairify to fetch vulnerability data by the image SHA.
func (c *Clairify) RetrieveImageDataBySHA(sha string, opts *types.GetImageDataOpts) (*v1.LayerEnvelope, error) {
	values := encodeValues(opts)
	request, err := http.NewRequest("GET", c.endpoint+"/scanner/sha/"+sha, nil)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = values.Encode()
	envelopeData, err := c.sendRequest(request, GetTimeout)
	if err != nil {
		return nil, err
	}
	var layerEnvelope v1.LayerEnvelope

	if err := easyjson.Unmarshal(envelopeData, &layerEnvelope); err != nil {
		return nil, err
	}
	return &layerEnvelope, err
}

// RetrieveImageDataByName contacts Clairify to fetch vulnerability data by the image name.
func (c *Clairify) RetrieveImageDataByName(image *types.Image, opts *types.GetImageDataOpts) (*v1.LayerEnvelope, error) {
	values := encodeValues(opts)
	url := fmt.Sprintf("%s/scanner/image/%s/%s/%s", c.endpoint, image.Registry, image.Remote, image.Tag)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	request.URL.RawQuery = values.Encode()
	envelopeData, err := c.sendRequest(request, GetTimeout)
	if err != nil {
		return nil, err
	}
	var layerEnvelope v1.LayerEnvelope
	if err := easyjson.Unmarshal(envelopeData, &layerEnvelope); err != nil {
		return nil, err
	}
	return &layerEnvelope, err
}

// GetVulnDefsMetadata contacts Clairify to fetch vulnerability definitions metadata.
func (c *Clairify) GetVulnDefsMetadata() (*protoV1.VulnDefsMetadata, error) {
	url := fmt.Sprintf("%s/scanner/vulndefs/metadata", c.endpoint)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	data, err := c.sendRequest(request, GetTimeout)
	if err != nil {
		return nil, err
	}

	var vulnDefsInfo protoV1.VulnDefsMetadata
	if err := json.Unmarshal(data, &vulnDefsInfo); err != nil {
		return nil, err
	}
	return &vulnDefsInfo, err
}
