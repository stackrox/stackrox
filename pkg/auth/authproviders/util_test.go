package authproviders

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeUIEndpoint(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected string
		wantErr  bool
	}{
		"already canonical host:port": {
			input:    "central.example.com:443",
			expected: "central.example.com:443",
		},
		"bare hostname keeps no port": {
			input:    "central.example.com",
			expected: "central.example.com",
		},
		"https scheme stripped, no port added": {
			input:    "https://central.example.com",
			expected: "central.example.com",
		},
		"https scheme with explicit port preserved": {
			input:    "https://central.example.com:8443",
			expected: "central.example.com:8443",
		},
		"http scheme stripped, no port added": {
			input:    "http://central.example.com",
			expected: "central.example.com",
		},
		"http scheme with explicit port preserved": {
			input:    "http://central.example.com:8080",
			expected: "central.example.com:8080",
		},
		"trailing slash stripped": {
			input:    "https://central.example.com/",
			expected: "central.example.com",
		},
		"path stripped": {
			input:    "https://central.example.com/some/path",
			expected: "central.example.com",
		},
		"trailing slash on bare host:port": {
			input:    "central.example.com:443/",
			expected: "central.example.com:443",
		},
		"localhost with port": {
			input:    "localhost:8000",
			expected: "localhost:8000",
		},
		"localhost without port": {
			input:    "localhost",
			expected: "localhost",
		},
		"IP address with port": {
			input:    "192.168.1.1:443",
			expected: "192.168.1.1:443",
		},
		"IP address without port": {
			input:    "192.168.1.1",
			expected: "192.168.1.1",
		},
		"https IP address": {
			input:    "https://192.168.1.1",
			expected: "192.168.1.1",
		},
		"custom port without scheme": {
			input:    "central.example.com:9443",
			expected: "central.example.com:9443",
		},
		"empty string": {
			wantErr: true,
		},
		"whitespace only": {
			input:   "   ",
			wantErr: true,
		},
		"trailing colon dropped": {
			input:    "central.example.com:",
			expected: "central.example.com",
		},
		"https trailing colon dropped": {
			input:    "https://central.example.com:",
			expected: "central.example.com",
		},
		"non-numeric port": {
			input:   "central.example.com:abc",
			wantErr: true,
		},
		"http localhost": {
			input:    "http://localhost:8080",
			expected: "localhost:8080",
		},
		"http loopback IP": {
			input:    "http://127.0.0.1:8080",
			expected: "127.0.0.1:8080",
		},
		"IPv6 loopback with brackets keeps no port": {
			input:    "[::1]",
			expected: "[::1]",
		},
		"IPv6 loopback with port": {
			input:    "[::1]:8080",
			expected: "[::1]:8080",
		},
		"https IPv6 loopback": {
			input:    "https://[::1]",
			expected: "[::1]",
		},
		"https IPv6 loopback with port": {
			input:    "https://[::1]:8443",
			expected: "[::1]:8443",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := NormalizeUIEndpoint(tc.input)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
