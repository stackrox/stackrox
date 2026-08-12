package vsockdialer

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestBuildWSURL(t *testing.T) {
	params := url.Values{}
	params.Set("port", "1024")
	params.Set("tls", "true")

	cases := map[string]struct {
		host    string
		wantURL string
		wantErr string // substring to match against the error; empty means no error expected
	}{
		"should keep https for client-go websocket RoundTripper": {
			host:    "https://api.example.com:6443",
			wantURL: "https://api.example.com:6443/apis/subresources.kubevirt.io/v1/namespaces/ns1/virtualmachineinstances/vm-a/vsock?port=1024&tls=true",
		},
		"should keep http for client-go websocket RoundTripper": {
			host:    "http://localhost:8080",
			wantURL: "http://localhost:8080/apis/subresources.kubevirt.io/v1/namespaces/ns1/virtualmachineinstances/vm-a/vsock?port=1024&tls=true",
		},
		"should reject unsupported scheme": {
			host:    "ftp://api.example.com",
			wantErr: "unsupported scheme",
		},
		"should reject unparseable host": {
			host:    "https://api.example.com:abc",
			wantErr: "parsing host",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := buildWSURL(&rest.Config{Host: tc.host}, "virtualmachineinstances", "ns1", "vm-a", "vsock", params)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.wantURL, got)
		})
	}
}
