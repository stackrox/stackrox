package types

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dummyTimeoutRoundTripper() promhttp.RoundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		return nil, os.ErrDeadlineExceeded
	}
}

func TestMetricsRoundTripperHappyPath(t *testing.T) {
	handler := NewMetricsHandler("happy")
	transport := handler.RoundTripper(http.DefaultTransport, "docker")
	client := &http.Client{Transport: transport}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
	resp, _ := client.Do(req)
	defer utils.IgnoreError(resp.Body.Close)

	assert.Equal(t, 1, handler.TestCollectRequestCounter(t))
	assert.Equal(t, 0, handler.TestCollectTimeoutCounter(t))
	assert.Equal(t, 1, handler.TestCollectHistogramCounter(t))
}

func TestMetricsRoundTripperTimeoutCounter(t *testing.T) {
	handler := NewMetricsHandler("timeout")

	transport := handler.RoundTripper(dummyTimeoutRoundTripper(), "docker")
	_, err := transport.RoundTrip(&http.Request{})
	require.Error(t, err)

	assert.Equal(t, 1, testutil.CollectAndCount(handler.timeoutCounter))
	assert.Equal(t, 1.0, testutil.ToFloat64(handler.timeoutCounter.WithLabelValues("docker")))
}
