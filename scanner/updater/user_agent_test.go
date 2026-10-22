package updater

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdaterUserAgent(t *testing.T) {
	for name, tc := range map[string]struct {
		version string
		want    string
	}{
		"release": {
			version: "5.1.0",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"v prefix": {
			version: "v5.1.0",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"git describe": {
			version: "5.1.0-12-g0123456789",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"dirty build": {
			version: "5.1.0-12-g0123456789-dirty",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"prerelease": {
			version: "5.1.0-rc.1",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"build metadata": {
			version: "5.1.0+private-build",
			want:    "Stackrox/Scanner/5.1.0",
		},
		"empty": {
			want: "Stackrox/Scanner/unknown",
		},
		"unrecognized": {
			version: "custom-build",
			want:    "Stackrox/Scanner/unknown",
		},
		"invalid suffix": {
			version: "5.1.0private",
			want:    "Stackrox/Scanner/unknown",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, updaterUserAgent(tc.version))
		})
	}
}

func TestUserAgentTransport(t *testing.T) {
	for name, headers := range map[string]http.Header{
		"nil headers": nil,
		"no user agent": {
			"Accept": {"application/xml"},
		},
		"ClairCore user agent": {
			"User-Agent": {"claircore/pkg/ovalutil.Fetcher"},
			"Accept":     {"application/xml"},
		},
		"multiple user agents": {
			"User-Agent": {"first", "second"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "https://example.com/feed", nil)
			require.NoError(t, err)
			req.Header = headers
			original := req.Header.Clone()
			response := &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			calls := 0
			tr := &userAgentTransport{
				userAgent: "Stackrox/Scanner/5.1.0",
				transport: httputil.RoundTripperFunc(func(got *http.Request) (*http.Response, error) {
					calls++
					assert.NotSame(t, req, got)
					assert.Equal(t, []string{"Stackrox/Scanner/5.1.0"}, got.Header.Values("User-Agent"))
					assert.Equal(t, req.Header.Get("Accept"), got.Header.Get("Accept"))
					assert.Equal(t, req.URL, got.URL)
					assert.Equal(t, req.Method, got.Method)
					assert.Equal(t, req.Context(), got.Context())
					return response, nil
				}),
			}
			got, err := tr.RoundTrip(req)
			require.NoError(t, err)
			assert.Same(t, response, got)
			assert.Equal(t, 1, calls)
			assert.Equal(t, original, req.Header)
		})
	}
}

func TestUserAgentTransportError(t *testing.T) {
	wantErr := errors.New("request failed")
	tr := &userAgentTransport{
		userAgent: "Stackrox/Scanner/5.1.0",
		transport: httputil.RoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return nil, wantErr
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.com/feed", nil)
	require.NoError(t, err)
	response, err := tr.RoundTrip(req)
	assert.Nil(t, response)
	assert.ErrorIs(t, err, wantErr)
}
