package docker

import (
	"crypto/tls"
	"strings"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil/proxy"
	"github.com/stackrox/rox/pkg/registries/types"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
)

// Config is the basic config for the docker registry.
type Config struct {
	// Endpoint defines the Docker Registry URL.
	Endpoint string
	// Insecure defines if the registry should be insecure.
	Insecure bool
	// DisableRepoList when true disables populating list of repos from remote registry.
	DisableRepoList bool

	// username defines the Username for the Docker Registry.
	username string
	// password defines the password for the Docker Registry.
	password string
	mutex    sync.RWMutex

	MetricsHandler *types.MetricsHandler
	// RegistryType is the underlying registry type as encoded in the image integration.
	RegistryType string
}

// GetCredentials returns the Docker basic auth credentials.
func (c *Config) GetCredentials() (string, string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.username, c.password
}

// SetCredentials sets the Docker basic auth credentials.
func (c *Config) SetCredentials(username string, password string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.username = username
	c.password = password
}

func (c *Config) formatURL() string {
	return FormatURL(c.Endpoint)
}

// FormatURL will return a formatted URL from a given registry endpoint.
func FormatURL(endpoint string) string {
	if strings.EqualFold(endpoint, "https://docker.io") || strings.EqualFold(endpoint, "docker.io") {
		endpoint = "https://registry-1.docker.io"
	}
	return urlfmt.FormatURL(endpoint, urlfmt.HTTPS, urlfmt.NoTrailingSlash)
}

// RegistryHostnameURL returns the hostname and url for a registry.
func RegistryHostnameURL(endpoint string) (string, string) {
	url := FormatURL(endpoint)

	host := urlfmt.GetServerFromURL(url)
	if strings.Contains(endpoint, "docker.io") {
		host = "docker.io"
	}

	return host, url
}

// DefaultTransport returns the default transport based on the configuration.
func DefaultTransport(cfg *Config) registry.Transport {
	transport := proxy.RoundTripper(
		proxy.WithDialTimeout(env.RegistryDialerTimeout.DurationSetting()),
		proxy.WithResponseHeaderTimeout(env.RegistryResponseTimeout.DurationSetting()),
	)
	if cfg.Insecure {
		transport = proxy.RoundTripper(
			proxy.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
			proxy.WithDialTimeout(env.RegistryDialerTimeout.DurationSetting()),
			proxy.WithResponseHeaderTimeout(env.RegistryResponseTimeout.DurationSetting()),
		)
	}
	transport = cfg.MetricsHandler.RoundTripper(transport, cfg.RegistryType)
	username, password := cfg.GetCredentials()
	return registry.WrapTransport(transport, strings.TrimSuffix(cfg.formatURL(), "/"), username, password)
}
