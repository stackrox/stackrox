package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProxyConfig(t *testing.T) {
	mustParse := func(s string) *url.URL {
		ret, err := url.Parse(s)
		if err != nil {
			t.Error(err)
		}
		return ret
	}
	tests := []struct {
		name    string
		input   string
		want    *url.URL
		wantErr string
	}{
		{
			name:    "empty",
			input:   "",
			want:    nil,
			wantErr: "",
		},
		{
			name:    "simple",
			input:   `url: http://localhost:8080/`,
			want:    mustParse("http://localhost:8080/"),
			wantErr: "",
		},
		{
			name:    "user",
			input:   "url: http://localhost/\nusername: User",
			want:    mustParse("http://User@localhost/"),
			wantErr: "",
		},
		{
			name:    "userpass",
			input:   "url: http://localhost/\nusername: User\npassword: hunter2\n",
			want:    mustParse("http://User:hunter2@localhost/"),
			wantErr: "",
		},
	}

	fakeReq, err := http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pc := proxyConfig{}
			err := yaml.Unmarshal([]byte(tt.input), &pc)
			if err != nil {
				t.Error(err)
			}
			err = pc.Validate()
			compiled := pc.Compile(environmentConfig{})
			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tt.wantErr)
			}
			var proxyURL *url.URL
			if compiled != nil {
				proxyURL, err = compiled.ProxyURL(fakeReq)
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, proxyURL)
		})
	}
}

func TestProxyExcludes(t *testing.T) {
	cases := []struct {
		OmitDefaultExcludes    bool
		Excludes               []string
		ProxyURLs, NoProxyURLs []string
	}{
		{
			NoProxyURLs: []string{"https://central.stackrox:1234", "http://localhost", "http://[::1]:1234", "https://foobar.local/bla"},
			ProxyURLs:   []string{"http://example.com", "http://www.example.com/bla"},
		},
		{
			Excludes:    []string{"*.excluded", "no.proxy"},
			NoProxyURLs: []string{"https://central.stackrox:1234", "http://localhost", "https://no.proxy", "https://foo.excluded/foo", "https://bar.excluded"},
			ProxyURLs:   []string{"http://example.com", "http://www.example.com/bla", "http://yes.proxy"},
		},
		{
			OmitDefaultExcludes: true,
			ProxyURLs:           []string{"https://central.stackrox:1234", "http://localhost.localdomain", "http://scanner.stackrox.svc:1234", "https://foobar.local/bla", "http://www.example.com"},
		},
		{
			OmitDefaultExcludes: true,
			Excludes:            []string{"*.excluded", "no.proxy"},
			NoProxyURLs:         []string{"https://no.proxy", "https://foo.excluded/foo", "https://bar.excluded"},
			ProxyURLs:           []string{"https://central.stackrox:1234", "http://localhost.localdomain", "http://scanner.stackrox.svc:1234", "https://foobar.local/bla", "http://yes.proxy/bla"},
		},
	}

	for i, testCase := range cases {
		tc := testCase
		t.Run(fmt.Sprintf("Case %d", i+1), func(t *testing.T) {
			cfg := proxyConfig{
				proxyEndpointConfig: proxyEndpointConfig{
					ProxyURL: "http://my.proxy:3128",
				},
				Excludes:            tc.Excludes,
				OmitDefaultExcludes: tc.OmitDefaultExcludes,
			}
			err := cfg.Validate()
			require.NoError(t, err)
			compiled := cfg.Compile(environmentConfig{})

			for _, u := range tc.NoProxyURLs {
				req, err := http.NewRequest(http.MethodGet, u, nil)
				require.NoError(t, err)
				proxyURL, err := compiled.ProxyURL(req)
				require.NoError(t, err)
				assert.Nilf(t, proxyURL, "Expected proxy not to be used for URL %v", u)
			}
			for _, u := range tc.ProxyURLs {
				req, err := http.NewRequest(http.MethodGet, u, nil)
				require.NoError(t, err)
				proxyURL, err := compiled.ProxyURL(req)
				require.NoError(t, err)
				assert.NotNilf(t, proxyURL, "Expected proxy to be used for URL %v", u)
			}
		})
	}
}
