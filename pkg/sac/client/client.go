package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/stackrox/default-authz-plugin/pkg/payload"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/httputil/proxy"
)

// Client is a simple interface describing retrieving some per user data from a separate service.
//go:generate mockgen-wrapper
type Client interface {
	ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) (allowed, denied []payload.AccessScope, err error)
}

type errorClient struct {
	err error
}

// New returns a new instance of Client.
func New(config *storage.HTTPEndpointConfig) (Client, error) {
	if err := validateEndpoint(config.GetEndpoint()); err != nil {
		return nil, err
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: config.GetSkipTlsVerify()}
	if config.GetCaCert() != "" {
		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(config.GetCaCert())); !ok {
			return nil, errors.New("no certificates found in PEM data")
		}
		tlsConfig.RootCAs = caCertPool
	}
	if config.GetClientCertPem() != "" || config.GetClientKeyPem() != "" {
		cert, err := tls.X509KeyPair([]byte(config.GetClientCertPem()), []byte(config.GetClientKeyPem()))
		if err != nil {
			return nil, errors.Wrap(err, "loading client certificate")
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}
	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
		Proxy:           proxy.FromConfig(),
	}
	client := &http.Client{Transport: transport}
	return &clientImpl{
		client: client,
		config: config,
	}, nil
}

// NewErrorClient returns an auth plugin client which will return an error for all auth requests
func NewErrorClient(err error) Client {
	return &errorClient{err: err}
}

func (ec *errorClient) ForUser(ctx context.Context, principal payload.Principal, scopes ...payload.AccessScope) (allowed, denied []payload.AccessScope, err error) {
	return nil, nil, ec.err
}

// The endpoint must be a valid URL and either use https or be localhost
func validateEndpoint(endpoint string) error {
	endpointURL, err := url.Parse(endpoint)
	if err != nil {
		return err
	}
	scheme := strings.ToLower(endpointURL.Scheme)
	if scheme == "https" {
		return nil
	}
	if scheme != "http" {
		return errors.Errorf("invalid scheme %q", scheme)
	}
	host := strings.ToLower(endpointURL.Hostname())
	if host == "localhost" {
		return nil
	}
	if strings.HasPrefix(host, "127.") {
		return nil
	}
	if host == "::1" {
		return nil
	}
	return errors.Errorf("invalid config: endpoint %s must start with https or be local", endpoint)
}
