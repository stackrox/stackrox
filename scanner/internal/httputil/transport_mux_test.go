package httputil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/buildinfo"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTransports() (http.RoundTripper, http.RoundTripper, http.RoundTripper) {
	defaultTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Default")),
		}, nil
	})
	centralTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Central")),
		}, nil
	})
	sensorTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Sensor")),
		}, nil
	})
	return defaultTransport, centralTransport, sensorTransport
}

func TestTransportMux(t *testing.T) {
	// This is currently not utilized in tests, but it will help us catch any changes to
	// both ROX_CENTRAL_ENDPOINT and ROX_SENSOR_ENDPOINT, as it'll change the default endpoints.
	t.Setenv(env.Namespace.EnvVar(), "something-else")

	defaultTransport, centralTransport, sensorTransport := testTransports()

	for _, testcase := range []struct {
		name string
		msg  string
		url  string
		envs map[string]string
	}{
		{
			// ROX_CENTRAL_ENDPOINT is not allowed to be empty, so we'll use the default value of
			// central.stackrox.svc:443 for this test.
			// This test will hopefully catch any changes and trigger the author of the change
			// to notify us to ensure we are all on the same page.
			name: "Central (default)",
			msg:  "Central",
			url:  "https://central.stackrox.svc/api/extensions/scannerdefinitions?file=repo2cpe",
			envs: map[string]string{
				env.CentralEndpoint.EnvVar(): "",
			},
		},
		{
			name: "Central (configured)",
			msg:  "Central",
			url:  "https://central.stackrox/api/extensions/scannerdefinitions?file=repo2cpe",
			envs: map[string]string{
				env.CentralEndpoint.EnvVar(): "central.stackrox",
			},
		},
		{
			// ROX_SENSOR_ENDPOINT is not allowed to be empty, so we'll use the default value of
			// sensor.stackrox.svc:443 for this test.
			// This test will hopefully catch any changes and trigger the author of the change
			// to notify us to ensure we are all on the same page.
			name: "Sensor (default)",
			msg:  "Sensor",
			url:  "https://sensor.stackrox.svc/api/extensions/scannerdefinitions?file=repo2cpe",
			envs: map[string]string{
				env.SensorEndpoint.EnvVar(): "",
			},
		},
		{
			name: "Sensor (configured)",
			msg:  "Sensor",
			url:  "https://sensor.rhacs/api/extensions/scannerdefinitions?file=repo2cpe",
			envs: map[string]string{
				env.SensorEndpoint.EnvVar(): "sensor.rhacs",
			},
		},
		{
			name: "Default",
			msg:  "Default",
			url:  "https://quay.io/image_layer_query_here",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			for k, v := range testcase.envs {
				t.Setenv(k, v)
			}

			tr, err := transportMux(defaultTransport, options{
				centralTransport: centralTransport,
				sensorTransport:  sensorTransport,
			})
			require.NoError(t, err)

			c := &http.Client{
				Transport: tr,
			}

			resp, err := c.Get(testcase.url)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = resp.Body.Close
			})
			msg, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, testcase.msg, string(msg))
		})
	}
}

func TestTransportMux_deny(t *testing.T) {
	// These are currently not utilized in tests, but it will help us catch any changes to
	// both ROX_CENTRAL_ENDPOINT and ROX_SENSOR_ENDPOINT, as it'll change the default endpoints.
	t.Setenv(env.Namespace.EnvVar(), "something-else")
	t.Setenv(env.CentralEndpoint.EnvVar(), "")
	t.Setenv(env.SensorEndpoint.EnvVar(), "")

	defaultTransport, _, _ := testTransports()

	tr, err := TransportMux(defaultTransport, WithDenyStackRoxServices(true))
	require.NoError(t, err)

	c := &http.Client{
		Transport: tr,
	}

	for _, testcase := range []struct {
		name      string
		url       string
		wantPanic bool
	}{
		{
			name:      "Central",
			url:       "https://central.stackrox.svc/api/extensions/scannerdefinitions?file=repo2cpe",
			wantPanic: true,
		},
		{
			name:      "Sensor",
			url:       "https://sensor.stackrox.svc/api/extensions/scannerdefinitions?file=repo2cpe",
			wantPanic: true,
		},
		{
			name:      "Default",
			url:       "https://quay.io/image_layer_query_here",
			wantPanic: false,
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
			if testcase.wantPanic {
				if buildinfo.ReleaseBuild {
					_, err := c.Get(testcase.url)
					assert.Error(t, err)
					return
				}
				assert.Panics(t, func() {
					_, _ = c.Get(testcase.url)
				})
				return
			}
			resp, err := c.Get(testcase.url)
			require.NoError(t, err)
			t.Cleanup(func() {
				_ = resp.Body.Close
			})
			msg, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, testcase.name, string(msg))
		})
	}
}
