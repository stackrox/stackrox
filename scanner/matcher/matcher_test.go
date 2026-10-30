package matcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/scanner/internal/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectDefaultTransport(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		releaseBuild          bool
		allowExternalVulnURLs bool
		expectDeny            bool
	}{
		"release build, external URLs not allowed (default)": {
			releaseBuild:          true,
			allowExternalVulnURLs: false,
			expectDeny:            true,
		},
		"release build, external URLs allowed": {
			releaseBuild:          true,
			allowExternalVulnURLs: true,
			expectDeny:            false,
		},
		"dev build, external URLs not allowed": {
			releaseBuild:          false,
			allowExternalVulnURLs: false,
			expectDeny:            false,
		},
		"dev build, external URLs allowed": {
			releaseBuild:          false,
			allowExternalVulnURLs: true,
			expectDeny:            false,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			transport := selectDefaultTransport(ctx, tt.releaseBuild, tt.allowExternalVulnURLs)
			require.NotNil(t, transport)

			if tt.expectDeny {
				req := httptest.NewRequest(http.MethodGet, "http://example.com/test", nil)
				if buildinfo.ReleaseBuild {
					resp, err := transport.RoundTrip(req)
					assert.Nil(t, resp, "expected nil response from DenyTransport")
					assert.Error(t, err, "expected error from DenyTransport")
					assert.Contains(t, err.Error(), "HTTP traffic denied")
				} else {
					assert.Panics(t, func() {
						_, _ = transport.RoundTrip(req)
					}, "expected DenyTransport to panic in dev builds")
				}
			} else {
				assert.Same(t, http.DefaultTransport, transport, "expected http.DefaultTransport")
			}
		})
	}
}

// TestTransportMuxIntegration tests the full transport selection + TransportMux chain
// using a local HTTP server. It verifies:
//   - External URLs are denied in release builds by default.
//   - External URLs are allowed when the opt-in is enabled.
//   - Sensor is always denied regardless of the opt-in.
//
// Central is denied in this test because we cannot set up mTLS certs in unit tests
// and Central routing is not the focus of this change.
func TestTransportMuxIntegration(t *testing.T) {
	ctx := context.Background()

	// Start a local HTTP server to act as an external host.
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "external-ok")
	}))
	defer localServer.Close()

	// Set Central and Sensor to known hostnames that don't collide with the local server.
	t.Setenv(env.CentralEndpoint.EnvVar(), "central.test.svc:443")
	t.Setenv(env.SensorEndpoint.EnvVar(), "sensor.test.svc:443")

	tests := map[string]struct {
		releaseBuild          bool
		allowExternalVulnURLs bool
		expectExternalAllowed bool
	}{
		"release default: external denied": {
			releaseBuild:          true,
			allowExternalVulnURLs: false,
			expectExternalAllowed: false,
		},
		"release opt-in: external allowed": {
			releaseBuild:          true,
			allowExternalVulnURLs: true,
			expectExternalAllowed: true,
		},
		"dev build: external always allowed": {
			releaseBuild:          false,
			allowExternalVulnURLs: false,
			expectExternalAllowed: true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			defaultTransport := selectDefaultTransport(ctx, tt.releaseBuild, tt.allowExternalVulnURLs)

			// Build TransportMux the same way NewMatcher does:
			// - Central denied (can't set up mTLS in unit test, not relevant here)
			// - Sensor always denied
			transport, err := httputil.TransportMux(
				defaultTransport,
				httputil.WithDenyStackRoxServices(true),
			)
			require.NoError(t, err)

			client := &http.Client{Transport: transport}

			// --- External URL (local HTTP server) ---
			if tt.expectExternalAllowed {
				resp, err := client.Get(localServer.URL + "/test-bundle")
				require.NoError(t, err)
				defer resp.Body.Close()
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, "external-ok", string(body), "external request should succeed")
			} else {
				assertDenied(t, func() error {
					_, err := client.Get(localServer.URL + "/test-bundle")
					return err
				}, "external request should be denied")
			}

			// --- Sensor is always denied ---
			assertDenied(t, func() error {
				_, err := client.Get("https://sensor.test.svc:443/some-endpoint")
				return err
			}, "sensor request should always be denied")
		})
	}
}

// assertDenied verifies that an HTTP call is denied by DenyTransport.
// In dev builds, DenyTransport panics via utils.ShouldErr.
// In release builds, it returns an error containing "HTTP traffic denied".
func assertDenied(t *testing.T, fn func() error, msgAndArgs ...interface{}) {
	t.Helper()
	if buildinfo.ReleaseBuild {
		err := fn()
		require.Error(t, err, msgAndArgs...)
		assert.Contains(t, err.Error(), "HTTP traffic denied", msgAndArgs...)
	} else {
		assert.Panics(t, func() {
			_ = fn()
		}, msgAndArgs...)
	}
}
