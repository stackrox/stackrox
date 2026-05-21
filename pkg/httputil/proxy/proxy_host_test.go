package proxy

import (
	"errors"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http/httpproxy"
)

func TestProxyHostForURL(t *testing.T) {
	prev, _ := globalProxyConfig.Load().(*compiledConfig)
	t.Cleanup(func() {
		if prev != nil {
			globalProxyConfig.Store(prev)
			return
		}
		globalProxyConfig.Store(defaultProxyConfig)
	})

	tests := map[string]struct {
		envCfg   environmentConfig
		endpoint string
		want     string
	}{
		"should return proxy host when https proxy matches": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{
					HTTPSProxy: "http://proxy.example.com:3128",
				},
			},
			endpoint: "https://central.example.com:443",
			want:     "proxy.example.com:3128",
		},
		"should return empty string when host is excluded by no proxy": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{
					HTTPSProxy: "http://proxy.example.com:3128",
					NoProxy:    "central.example.com",
				},
			},
			endpoint: "https://central.example.com:443",
			want:     "",
		},
		"should return empty string when no proxy is configured": {
			envCfg:   environmentConfig{},
			endpoint: "https://central.example.com:443",
			want:     "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			globalProxyConfig.Store((&proxyConfig{OmitDefaultExcludes: true}).Compile(tc.envCfg))
			assert.Equal(t, tc.want, ProxyHostForURL(tc.endpoint))
		})
	}

	t.Run("should return empty string when request creation fails", func(t *testing.T) {
		assert.Equal(t, "", ProxyHostForURL("://bad-url"))
	})

	t.Run("should return empty string when proxy resolution fails", func(t *testing.T) {
		globalProxyConfig.Store(&compiledConfig{
			httpsFunc: func(*url.URL) (*url.URL, error) {
				return nil, errors.New("proxy lookup failed")
			},
		})

		assert.Equal(t, "", ProxyHostForURL("https://central.example.com:443"))
	})
}
