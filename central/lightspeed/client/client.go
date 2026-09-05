package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/tlsutils"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	defaultRequestTimeout = 30 * time.Second
)

var (
	// LightspeedEndpoint is the URL of the OpenShift Lightspeed API. URL will be populated from integration in a follow up PR
	LightspeedEndpoint = env.RegisterSetting("ROX_LIGHTSPEED_ENDPOINT",
		env.WithDefault("https://lightspeed-app-server.openshift-lightspeed.svc:8443"))
)

// QueryRequest is the request payload for the OLS query API.
type QueryRequest struct {
	Query   string `json:"query"`
	Context string `json:"context,omitempty"`
}

// QueryResponse is the response payload from the OLS query API.
type QueryResponse struct {
	Response string `json:"response"`
}

// Client calls the OpenShift Lightspeed API.
type Client interface {
	Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error)
	TestConnectivity() error
}

// NewClient creates a Client that authenticates with the OLS API using the
// pod's Kubernetes service account token and trusts the service CA.
func NewClient() Client {
	transport := tlsutils.TransportWithServiceCA()
	transport.Proxy = proxy.FromConfig()

	return &clientImpl{
		loadToken: satoken.LoadTokenFromFile,
		endpoint:  LightspeedEndpoint.Setting(),
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   defaultRequestTimeout,
		},
	}
}

type clientImpl struct {
	loadToken  func() (string, error)
	endpoint   string
	httpClient *http.Client
}

func (c *clientImpl) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	token, err := c.loadToken()
	if err != nil {
		return nil, errors.Wrap(err, "loading service account token")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling request")
	}

	url := fmt.Sprintf("%s/v1/query", c.endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, errors.Wrap(err, "building HTTP request")
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, errors.Wrap(err, "sending request to Lightspeed")
	}
	defer utils.IgnoreError(resp.Body.Close)

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Lightspeed API returned HTTP %d", resp.StatusCode)
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "decoding Lightspeed response")
	}

	return &result, nil
}

// TestConnectivity checks if the Lightspeed service is reachable.
func (c *clientImpl) TestConnectivity() error {
	healthURL := fmt.Sprintf("%s/readiness", c.endpoint)
	resp, err := c.httpClient.Get(healthURL)
	if err != nil {
		return errors.Wrap(err, "failed to connect to AI service")
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("AI service returned status %d", resp.StatusCode)
	}

	return nil
}
