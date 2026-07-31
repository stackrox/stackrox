package retryablehttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestConfigureRESTConfig_Defaults(t *testing.T) {
	restCfg := &rest.Config{}

	ConfigureRESTConfig(restCfg)

	assert.NotNil(t, restCfg.WrapTransport)
}

func TestConfigureRESTConfig_CustomTimeout(t *testing.T) {
	customTimeout := 60 * time.Second
	restCfg := &rest.Config{
		Timeout: customTimeout,
	}

	ConfigureRESTConfig(restCfg)

	assert.Equal(t, customTimeout, restCfg.Timeout)
}

func TestConfigureRESTConfig_WithOptions(t *testing.T) {
	testLogger := logging.ModuleForName("test").Logger()
	restCfg := &rest.Config{}

	ConfigureRESTConfig(restCfg,
		WithLogger(NewDebugLogger(testLogger)),
		WithRetryMax(5),
		WithRetryWaitMax(5*time.Second),
		WithRetryWaitMin(1*time.Second),
	)

	assert.NotNil(t, restCfg.WrapTransport)
}

func TestConfigureRESTConfig_PreservesExistingWrapTransport(t *testing.T) {
	existingWrapperCalled := false

	restCfg := &rest.Config{
		Timeout: 30 * time.Second,
	}

	restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		existingWrapperCalled = true
		return rt
	}

	ConfigureRESTConfig(restCfg)

	// Call the configured WrapTransport to verify chaining.
	mockTransport := &mockRoundTripper{}
	wrappedTransport := restCfg.WrapTransport(mockTransport)

	// Verify the existing wrapper was called.
	assert.True(t, existingWrapperCalled, "existing WrapTransport should be preserved and called")
	assert.NotNil(t, wrappedTransport)
}

func TestConfigureRESTConfig_NilExistingWrapTransport(t *testing.T) {
	restCfg := &rest.Config{
		Timeout: 30 * time.Second,
	}

	restCfg.WrapTransport = nil

	ConfigureRESTConfig(restCfg)

	// Verify WrapTransport was configured even when starting from nil.
	assert.NotNil(t, restCfg.WrapTransport)

	// Call the configured WrapTransport to verify it works.
	mockTransport := &mockRoundTripper{}
	wrappedTransport := restCfg.WrapTransport(mockTransport)
	assert.NotNil(t, wrappedTransport)
}

func TestWithRetryMax(t *testing.T) {
	cfg := &config{
		retryMax: defaultRetryMax,
	}

	opt := WithRetryMax(10)
	opt(cfg)

	assert.Equal(t, 10, cfg.retryMax)
}

func TestWithRetryWaitMin(t *testing.T) {
	cfg := &config{
		retryWaitMin: defaultRetryWaitMin,
	}

	opt := WithRetryWaitMin(2 * time.Second)
	opt(cfg)

	assert.Equal(t, 2*time.Second, cfg.retryWaitMin)
}

func TestWithRetryWaitMax(t *testing.T) {
	cfg := &config{
		retryWaitMax: defaultRetryWaitMax,
	}

	opt := WithRetryWaitMax(10 * time.Second)
	opt(cfg)

	assert.Equal(t, 10*time.Second, cfg.retryWaitMax)
}

func TestWithResponseHeaderTimeout(t *testing.T) {
	cfg := &config{}

	opt := WithResponseHeaderTimeout(15 * time.Second)
	opt(cfg)

	assert.Equal(t, 15*time.Second, cfg.responseHeaderTimeout)
}

func TestConfigureRESTConfig_RetriesOnHangingServer(t *testing.T) {
	var attempts atomic.Int32

	// Server that hangs (never sends headers) on the first 2 requests
	// and succeeds on the 3rd.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n <= 2 {
			<-r.Context().Done()
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	restCfg := &rest.Config{
		Host:    srv.URL,
		Timeout: 5 * time.Second,
	}
	ConfigureRESTConfig(restCfg,
		WithRetryMax(3),
		WithRetryWaitMin(0),
		WithRetryWaitMax(0),
		WithResponseHeaderTimeout(1*time.Second),
	)

	transport, err := rest.TransportFor(restCfg)
	require.NoError(t, err)
	client := &http.Client{Transport: transport}

	ctx, cancel := context.WithTimeout(context.Background(), restCfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Attempts seen by server: %d", attempts.Load())
		t.Fatalf("Request should have succeeded on attempt 3 but failed: %v", err)
	}
	resp.Body.Close()

	assert.Equal(t, int32(3), attempts.Load(),
		"expected exactly 3 attempts: 2 header timeouts then 1 success")
}

func TestConfigureRESTConfig_SlowBodyNotInterrupted(t *testing.T) {
	// Server that sends headers immediately but streams the body slowly.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		require.True(t, ok)

		// Stream body over 2s, well past the 500ms header timeout.
		for i := range 4 {
			fmt.Fprintf(w, "chunk %d\n", i)
			flusher.Flush()
			time.Sleep(500 * time.Millisecond)
		}
	}))
	defer srv.Close()

	restCfg := &rest.Config{
		Host:    srv.URL,
		Timeout: 10 * time.Second,
	}
	ConfigureRESTConfig(restCfg,
		WithRetryMax(3),
		WithResponseHeaderTimeout(500*time.Millisecond),
	)

	transport, err := rest.TransportFor(restCfg)
	require.NoError(t, err)
	client := &http.Client{Transport: transport}

	ctx, cancel := context.WithTimeout(context.Background(), restCfg.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err, "request should succeed — headers arrived before timeout")
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "body read should complete — header timeout does not affect body transfer")
	assert.True(t, strings.Contains(string(body), "chunk 3"),
		"expected all chunks to be received, got: %s", string(body))
}

// mockRoundTripper is a simple mock implementation of http.RoundTripper for testing
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}
