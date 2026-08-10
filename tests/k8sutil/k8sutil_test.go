//go:build test && !test_e2e && !test_e2e_vm

package k8sutil

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestCreateK8sClientWithConfig_RetriesOnFailure(t *testing.T) {
	var callCount int

	mockTransport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		callCount++
		currentCall := callCount

		t.Logf("Mock transport call #%d to %s", currentCall, r.URL.String())

		if currentCall <= 2 {
			return nil, errors.New("network error: connection refused")
		}

		responseBody := `{
			"major": "1",
			"minor": "28",
			"gitVersion": "v1.28.0",
			"gitCommit": "abcd1234",
			"buildDate": "2023-08-01T12:00:00Z",
			"goVersion": "go1.20.6",
			"compiler": "gc",
			"platform": "linux/amd64"
		}`

		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader(responseBody)),
			Header:     make(http.Header),
		}, nil
	})

	restCfg := &rest.Config{
		Host: "https://mock-k8s-api.example.com",
		WrapTransport: func(http.RoundTripper) http.RoundTripper {
			return mockTransport
		},
	}
	ConfigureRetryableTransport(t, restCfg)

	client := CreateK8sClientWithConfig(t, restCfg)
	require.NotNil(t, client, "client should not be nil")

	version, err := client.Discovery().ServerVersion()

	require.NoError(t, err, "Discovery call should succeed after retries")
	require.NotNil(t, version, "Server version should not be nil")
	assert.Equal(t, "1", version.Major)
	assert.Equal(t, "28", version.Minor)

	assert.Equal(t, 3, callCount, "Should have made exactly 3 calls (2 retries + 1 success)")
	t.Logf("Successfully completed after %d calls (including retries)", callCount)
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return fn(r)
}
