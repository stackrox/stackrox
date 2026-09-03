package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/satoken"
	"github.com/stackrox/rox/pkg/utils"
)

const (
	serviceOperatorCAPath = "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt"
	defaultRequestTimeout = 30 * time.Second
)

var log = logging.LoggerForModule()

// EndpointResolver provides the AI integration endpoint. If no integration
// is configured it returns ("", false, nil) so the client can fall back to
// the environment variable.
type EndpointResolver interface {
	GetEndpoint(ctx context.Context) (string, bool, error)
}

// QueryRequest is the request payload for the OLS query API.
type QueryRequest struct {
	Query string `json:"query"`
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
// The resolver is used to look up the endpoint from a stored AI integration.
func NewClient(resolver EndpointResolver) Client {
	return &clientImpl{
		loadToken: satoken.LoadTokenFromFile,
		resolver:  resolver,
		httpClient: &http.Client{
			Transport: transportWithServiceCA(),
			Timeout:   defaultRequestTimeout,
		},
	}
}

type clientImpl struct {
	loadToken  func() (string, error)
	resolver   EndpointResolver
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

func (c *clientImpl) resolveEndpoint(ctx context.Context) (string, error) {
	if c.resolver != nil {
		if endpoint, exists, err := c.resolver.GetEndpoint(ctx); err != nil {
			return "", errors.Wrap(err, "resolving AI integration endpoint")
		} else if exists && endpoint != "" {
			return endpoint, nil
		}
	}
	return "", errors.New("AI integration not configured")
}

func (c *clientImpl) Query(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	endpoint, err := c.resolveEndpoint(ctx)
	if err != nil {
		return nil, err
	}

	token, err := c.loadToken()
	if err != nil {
		return nil, errors.Wrap(err, "loading service account token")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrap(err, "marshaling request")
	}

	url := fmt.Sprintf("%s/v1/query", endpoint)
	log.Debugf("OLS request URL: %s, body length: %d bytes", url, len(body))
	log.Debugf("OLS request body: %s", string(body))

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
		respBody, _ := io.ReadAll(resp.Body)
		log.Errorf("OLS returned HTTP %d, response body: %s", resp.StatusCode, string(respBody))
		return nil, fmt.Errorf("Lightspeed API returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, errors.Wrap(err, "decoding Lightspeed response")
	}

	return &result, nil
}
