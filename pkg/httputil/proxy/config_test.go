package proxy

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http/httpproxy"
)

func TestCompileProxyFallbackForNonStandardSchemes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "198.51.100.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	const (
		httpsProxy = "http://https.proxy:3128"
		httpProxy  = "http://http.proxy:3128"
		allProxy   = "http://all.proxy:3128"
	)

	mustParseURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	cases := map[string]struct {
		envCfg        environmentConfig
		reqURL        string
		expectedProxy *url.URL
	}{
		"tcp scheme uses ALL_PROXY when set": {
			envCfg: environmentConfig{
				Config:   httpproxy.Config{HTTPSProxy: httpsProxy, HTTPProxy: httpProxy},
				AllProxy: allProxy,
			},
			reqURL:        "tcp://quay.io:443",
			expectedProxy: mustParseURL(allProxy),
		},
		"tcp scheme falls back to HTTPS_PROXY when ALL_PROXY is not set": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{HTTPSProxy: httpsProxy, HTTPProxy: httpProxy},
			},
			reqURL:        "tcp://quay.io:443",
			expectedProxy: mustParseURL(httpsProxy),
		},
		"tcp scheme falls back to HTTP_PROXY via HTTPS fallback chain": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{HTTPProxy: httpProxy},
			},
			reqURL:        "tcp://quay.io:443",
			expectedProxy: mustParseURL(httpProxy),
		},
		"tcp scheme returns nil when no proxy is configured": {
			envCfg:        environmentConfig{},
			reqURL:        "tcp://quay.io:443",
			expectedProxy: nil,
		},
		"http scheme uses HTTP_PROXY unaffected by change": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{HTTPProxy: httpProxy, HTTPSProxy: httpsProxy},
			},
			reqURL:        "http://example.com",
			expectedProxy: mustParseURL(httpProxy),
		},
		"https scheme uses HTTPS_PROXY unaffected by change": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{HTTPProxy: httpProxy, HTTPSProxy: httpsProxy},
			},
			reqURL:        "https://example.com",
			expectedProxy: mustParseURL(httpsProxy),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := proxyConfig{OmitDefaultExcludes: true}
			compiled := cfg.Compile(tc.envCfg)

			req, err := http.NewRequest(http.MethodGet, tc.reqURL, nil)
			require.NoError(t, err)

			proxyURL, err := compiled.ProxyURL(req)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedProxy, proxyURL)
		})
	}
}

func TestCompileEnvVarsNotPolluted(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "198.51.100.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	const (
		httpsProxy = "http://https.proxy:3128"
		httpProxy  = "http://http.proxy:3128"
		allProxy   = "http://all.proxy:3128"
	)

	cases := map[string]struct {
		envCfg           environmentConfig
		expectedAllProxy string
	}{
		"all_proxy reflects ALL_PROXY when explicitly set": {
			envCfg: environmentConfig{
				Config:   httpproxy.Config{HTTPSProxy: httpsProxy, HTTPProxy: httpProxy},
				AllProxy: allProxy,
			},
			expectedAllProxy: allProxy,
		},
		"all_proxy is empty when ALL_PROXY is not set even though HTTPS_PROXY is": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{HTTPSProxy: httpsProxy, HTTPProxy: httpProxy},
			},
			expectedAllProxy: "",
		},
		"all_proxy is empty when no proxy is configured": {
			envCfg:           environmentConfig{},
			expectedAllProxy: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := proxyConfig{OmitDefaultExcludes: true}
			compiled := cfg.Compile(tc.envCfg)

			assert.Equal(t, tc.expectedAllProxy, compiled.envVars["all_proxy"],
				"all_proxy env var must only reflect explicitly configured ALL_PROXY, never a fallback value")
		})
	}
}

func TestCompileTCPFallbackWithDefaultExcludes(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "198.51.100.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	const httpsProxy = "http://https.proxy:3128"

	cfg := proxyConfig{OmitDefaultExcludes: false}
	compiled := cfg.Compile(environmentConfig{
		Config: httpproxy.Config{HTTPSProxy: httpsProxy},
	})

	mustParseURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	req, err := http.NewRequest(http.MethodGet, "tcp://quay.io:443", nil)
	require.NoError(t, err)
	proxyURL, err := compiled.ProxyURL(req)
	require.NoError(t, err)
	assert.Equal(t, mustParseURL(httpsProxy), proxyURL,
		"tcp:// fallback must work when OmitDefaultExcludes is false")

	reqExcluded, err := http.NewRequest(http.MethodGet, "tcp://central.stackrox:443", nil)
	require.NoError(t, err)
	proxyURL, err = compiled.ProxyURL(reqExcluded)
	require.NoError(t, err)
	assert.Nil(t, proxyURL,
		"default excludes must still apply for tcp:// scheme")
}

func TestCompileYAMLConfigDrivenProxy(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "198.51.100.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	const (
		yamlHTTPProxy  = "http://yaml-http.proxy:3128"
		yamlHTTPSProxy = "http://yaml-https.proxy:3128"
	)

	mustParseURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	cfg := proxyConfig{
		HTTP:                proxyEndpointConfig{ProxyURL: yamlHTTPProxy},
		HTTPS:               proxyEndpointConfig{ProxyURL: yamlHTTPSProxy},
		OmitDefaultExcludes: true,
	}
	compiled := cfg.Compile(environmentConfig{})

	t.Run("https scheme uses YAML HTTPS config", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "https://example.com", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Equal(t, mustParseURL(yamlHTTPSProxy), proxyURL)
	})

	t.Run("http scheme uses YAML HTTP config", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Equal(t, mustParseURL(yamlHTTPProxy), proxyURL)
	})

	t.Run("tcp scheme falls back to YAML HTTPS config", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "tcp://quay.io:443", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Equal(t, mustParseURL(yamlHTTPSProxy), proxyURL,
			"tcp:// must fall back to HTTPS proxy from YAML config via c.HTTPS")
	})
}

func TestCompileNoProxyExcludesTCPScheme(t *testing.T) {
	t.Setenv("KUBERNETES_SERVICE_HOST", "198.51.100.1")
	t.Setenv("KUBERNETES_SERVICE_PORT", "443")

	const httpsProxy = "http://https.proxy:3128"

	mustParseURL := func(s string) *url.URL {
		u, err := url.Parse(s)
		require.NoError(t, err)
		return u
	}

	cfg := proxyConfig{OmitDefaultExcludes: true}
	compiled := cfg.Compile(environmentConfig{
		Config: httpproxy.Config{
			HTTPSProxy: httpsProxy,
			NoProxy:    "quay.io,internal.example.com",
		},
	})

	t.Run("tcp scheme excluded by NO_PROXY", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "tcp://quay.io:443", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Nil(t, proxyURL,
			"NO_PROXY exclusions must apply to tcp:// scheme URLs")
	})

	t.Run("tcp scheme not excluded uses proxy", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "tcp://registry.redhat.io:443", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Equal(t, mustParseURL(httpsProxy), proxyURL,
			"tcp:// to non-excluded host must use the proxy")
	})

	t.Run("tcp scheme excluded by wildcard NO_PROXY", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "tcp://foo.internal.example.com:443", nil)
		require.NoError(t, err)
		proxyURL, err := compiled.ProxyURL(req)
		require.NoError(t, err)
		assert.Nil(t, proxyURL,
			"NO_PROXY subdomain exclusion must apply to tcp:// scheme")
	})
}
