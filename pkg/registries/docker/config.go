package docker

import (
	"crypto/tls"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Config is the basic config for the docker registry.
type Config struct {
	// Endpoint defines the Docker Registry URL.
	Endpoint string
	// Username defines the Username for the Docker Registry.
	Username string
	// Password defines the password for the Docker Registry.
	Password string
	// Insecure defines if the registry should be insecure.
	Insecure bool
	// DisableRepoList when true disables populating list of repos from remote registry.
	DisableRepoList bool
	// Transport defines a transport for authenticating to the Docker registry.
	Transport registry.Transport
}

func (c *Config) formatURL() string {
	endpoint := c.Endpoint
	if strings.EqualFold(endpoint, "https://docker.io") || strings.EqualFold(endpoint, "docker.io") {
		endpoint = "https://registry-1.docker.io"
	}
	return urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
}

// GetTransport returns the transport which provides authentication to the Docker registry.
// Returns `Config.Transport` if it is set. Otherwise returns a default transport.
func (c *Config) GetTransport() registry.Transport {
	if c.Transport != nil {
		return c.Transport
	}
	return DefaultTransport(c)
}

// DefaultTransport returns the default transport based on the configuration.
func DefaultTransport(cfg *Config) registry.Transport {
	transport := proxy.RoundTripper()
	if cfg.Insecure {
		transport = proxy.RoundTripperWithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	return registry.WrapTransport(transport, strings.TrimSuffix(cfg.formatURL(), "/"), cfg.Username, cfg.Password)
}
