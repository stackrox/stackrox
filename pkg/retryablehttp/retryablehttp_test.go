package retryablehttp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stretchr/testify/assert"
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

// mockRoundTripper is a simple mock implementation of http.RoundTripper for testing
type mockRoundTripper struct{}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}
