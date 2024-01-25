package httputil

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportMux(t *testing.T) {
	t.Setenv(env.Namespace.EnvVar(), "rhacs")

	defaultTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "quay.io", req.URL.Host)
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Default")),
		}, nil
	})
	centralTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "central.rhacs.svc", req.URL.Host)
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Central")),
		}, nil
	})
	sensorTransport := httputil.RoundTripperFunc(func(req *http.Request) (*http.Response, error) {
		assert.Equal(t, "sensor.rhacs.svc", req.URL.Host)
		return &http.Response{
			Body: io.NopCloser(strings.NewReader("Sensor")),
		}, nil
	})

	tr, err := transportMux(defaultTransport, options{
		centralTransport: centralTransport,
		sensorTransport:  sensorTransport,
	})
	require.NoError(t, err)

	c := &http.Client{
		Transport: tr,
	}

	for _, testcase := range []struct {
		name string
		url  string
	}{
		{
			name: "Central",
			url:  "https://central.rhacs.svc/api/extensions/scannerdefinitions?file=repo2cpe",
		},
		{
			name: "Sensor",
			url:  "https://sensor.rhacs.svc/api/extensions/scannerdefinitions?file=repo2cpe",
		},
		{
			name: "Default",
			url:  "https://quay.io/image_layer_query_here",
		},
	} {
		t.Run(testcase.name, func(t *testing.T) {
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
