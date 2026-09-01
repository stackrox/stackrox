package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	serviceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
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
}

// NewClient creates a Client that authenticates with the OLS API using the
// pod's Kubernetes service account token and trusts the service CA.
func NewClient() Client {
	return &clientImpl{
		loadToken: satoken.LoadTokenFromFile,
		endpoint:  LightspeedEndpoint.Setting(),
		httpClient: &http.Client{
			Transport: transportWithServiceCA(),
			Timeout:   defaultRequestTimeout,
		},
	}
}

type clientImpl struct {
	loadToken  func() (string, error)
	endpoint   string
	httpClient *http.Client
}

func transportWithServiceCA() http.RoundTripper {
	rootCAs, err := x509.SystemCertPool()
	if err != nil {
		rootCAs = x509.NewCertPool()
	}

	if serviceCA, err := os.ReadFile(serviceOperatorCAPath); err == nil {
		rootCAs.AppendCertsFromPEM(serviceCA)
	}

	return &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: rootCAs,
		},
		Proxy: proxy.FromConfig(),
	}
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
