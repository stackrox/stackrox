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
		wantHost string
		wantErr  string
	}{
		"should return proxy host when https proxy matches": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{
					HTTPSProxy: "http://proxy.example.com:3128",
				},
			},
			endpoint: "https://central.example.com:443",
			wantHost: "proxy.example.com:3128",
		},
		"should return empty string when host is excluded by no proxy": {
			envCfg: environmentConfig{
				Config: httpproxy.Config{
					HTTPSProxy: "http://proxy.example.com:3128",
					NoProxy:    "central.example.com",
				},
			},
			endpoint: "https://central.example.com:443",
			wantHost: "",
		},
		"should return empty string when no proxy is configured": {
			envCfg:   environmentConfig{},
			endpoint: "https://central.example.com:443",
			wantHost: "",
		},
		"should return error when request creation fails": {
			endpoint: "://bad-url",
			wantErr:  `parse "://bad-url": missing protocol scheme`,
		},
		"should return error when proxy resolution fails": {
			endpoint: "https://central.example.com:443",
			wantErr:  "proxy lookup failed",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.wantErr == "proxy lookup failed" {
				globalProxyConfig.Store(&compiledConfig{
					httpsFunc: func(*url.URL) (*url.URL, error) {
						return nil, errors.New("proxy lookup failed")
					},
				})
			} else {
				globalProxyConfig.Store((&proxyConfig{OmitDefaultExcludes: true}).Compile(tc.envCfg))
			}

			gotHost, err := ProxyHostForURL(tc.endpoint)
			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.wantErr)
			}
			assert.Equal(t, tc.wantHost, gotHost)
		})
	}
}
