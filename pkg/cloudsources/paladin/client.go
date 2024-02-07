package paladin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/clientconn"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/urlfmt"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/version"
)

const (
	// Time format for Paladin API timestamps.
	timeFormat = "2006-01-02 15:04:00+0000"
)

// AssetsResponse holds the response returned by the Paladin Cloud API.
type AssetsResponse struct {
	Assets []Asset `json:"assets,omitempty"`
}

// Asset holds the asset as returned by the Paladin Cloud API.
type Asset struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Type              string    `json:"type"`
	Source            string    `json:"source"`
	Region            string    `json:"region"`
	FirstDiscoveredAt time.Time `json:"firstDiscoveryDate"`
}

// UnmarshalJSON ummarshals Paladin Cloud API responses to the Asset struct.
// Paladin API uses a custom time format which leads to errors when unmarshalling JSON without
// customizations.
// Hence, need to use an intermediary struct, parse the time according to the layout, and then
// fill the asset struct accordingly.
func (a *Asset) UnmarshalJSON(b []byte) error {
	var intermediary struct {
		ID                string `json:"id"`
		Name              string `json:"name"`
		Type              string `json:"type"`
		Source            string `json:"source"`
		Region            string `json:"region"`
		FirstDiscoveredAt string `json:"firstDiscoveryDate"`
	}
	if err := json.Unmarshal(b, &intermediary); err != nil {
		return err
	}

	discoveredAt, err := time.Parse(timeFormat, intermediary.FirstDiscoveredAt)
	if err != nil {
		return err
	}

	*a = Asset{
		ID:                intermediary.ID,
		Name:              intermediary.Name,
		Type:              intermediary.Type,
		Source:            intermediary.Source,
		Region:            intermediary.Region,
		FirstDiscoveredAt: discoveredAt,
	}
	return nil
}

// Client can be used to interact with the Paladin Cloud API.
type Client struct {
	httpClient *http.Client
	endpoint   string
}

// paladinTransportWrapper adds auth information to the underlying transport as well as the user agent.
type paladinTransportWrapper struct {
	baseTransport http.RoundTripper
	token         string
	acsVersion    string
}

func (p *paladinTransportWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	// The Paladin Cloud API expects the Authorization header in the format '"Authorization:" <TOKEN>'
	// instead of e.g. Bearer token format.
	req.Header.Add("Authorization", p.token)
	req.Header.Add("User-Agent", clientconn.GetUserAgent())
	return p.baseTransport.RoundTrip(req)
}

// NewClient creates a client to interact with Paladin Cloud APIs.
func NewClient(cfg *storage.CloudSource) *Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.HTTPClient.Transport = &paladinTransportWrapper{
		baseTransport: proxy.RoundTripper(),
		token:         cfg.GetCredentials().GetSecret(),
		acsVersion:    version.GetMainVersion(),
	}
	retryClient.HTTPClient.Timeout = 30 * time.Second
	retryClient.RetryWaitMin = 10 * time.Second

	return &Client{
		httpClient: retryClient.StandardClient(),
		endpoint:   urlfmt.FormatURL(cfg.GetPaladinCloud().GetEndpoint(), urlfmt.HTTPS, urlfmt.NoTrailingSlash),
	}
}

// GetAssets returns the discovered assets from Paladin Cloud.
func (c *Client) GetAssets(ctx context.Context) (*AssetsResponse, error) {
	var assets AssetsResponse

	if err := c.sendRequest(ctx, http.MethodGet, "/v2/assets", "?category=k8s", &assets); err != nil {
		return nil, errors.Wrap(err, "retrieving assets")
	}

	return &assets, nil
}

func (c *Client) sendRequest(ctx context.Context, method string, apiPath string, query string, response interface{}) error {
	path, err := url.JoinPath(c.endpoint, apiPath)
	if err != nil {
		return errox.InvalidArgs.CausedBy(err)
	}

	req, err := http.NewRequestWithContext(ctx, method, path+query, nil)
	if err != nil {
		return errors.Wrap(err, "creating request")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "executing request")
	}

	defer utils.IgnoreError(resp.Body.Close)

	if !httputil.Is2xxStatusCode(resp.StatusCode) {
		return getErrorResponse(resp)
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return errors.Wrap(err, "decoding response body")
	}

	return nil
}

func getErrorResponse(resp *http.Response) error {
	buf := &bytes.Buffer{}
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return fmt.Errorf("request failed with status %d", resp.StatusCode)
	}
	return fmt.Errorf("request failed with status %d and response %q", resp.StatusCode, buf.String())
}
