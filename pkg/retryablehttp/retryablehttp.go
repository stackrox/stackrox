package retryablehttp

import (
	"net/http"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/stackrox/rox/pkg/logging"
	"k8s.io/client-go/rest"
)

var log = logging.ModuleForName("retryablehttp").Logger()

const (
	defaultRetryMax     = 3
	defaultRetryWaitMin = 500 * time.Millisecond
	defaultRetryWaitMax = 2 * time.Second
)

type config struct {
	logger       retryablehttp.Logger
	retryMax     int
	retryWaitMax time.Duration
	retryWaitMin time.Duration
}

// Option configures retry behavior for HTTP transports.
type Option func(*config)

// WithLogger sets a custom logger for retry operations.
// Default is the module-specific logger for retryablehttp.
func WithLogger(logger retryablehttp.Logger) Option {
	return func(c *config) {
		c.logger = logger
	}
}

// WithRetryMax sets the maximum number of retry attempts.
// Default is 3 retries.
func WithRetryMax(max int) Option {
	return func(c *config) {
		c.retryMax = max
	}
}

// WithRetryWaitMax sets the maximum wait time between retry attempts.
// Default is 2 seconds.
func WithRetryWaitMax(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMax = d
	}
}

// WithRetryWaitMin sets the minimum wait time between retry attempts.
// Default is 500 milliseconds.
func WithRetryWaitMin(d time.Duration) Option {
	return func(c *config) {
		c.retryWaitMin = d
	}
}

// ConfigureRESTConfig wraps a Kubernetes REST config's transport with retry logic.
// This adds automatic retry for transient network errors, making Kubernetes clients more resilient.
//
// The function preserves any existing WrapTransport configuration by chaining the retryable
// transport after existing transport wrappers.
//
// Example usage:
//
//	restCfg, _ := rest.InClusterConfig()
//	retryablehttp.ConfigureRESTConfig(restCfg)
//	client, _ := kubernetes.NewForConfig(restCfg)
func ConfigureRESTConfig(restCfg *rest.Config, opts ...Option) {
	cfg := &config{
		logger:       NewDebugLogger(log),
		retryMax:     defaultRetryMax,
		retryWaitMax: defaultRetryWaitMax,
		retryWaitMin: defaultRetryWaitMin,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Set retryable client timeout to a fraction of REST config timeout.
	// This ensures we have time for retries before the overall timeout expires.
	clientTimeout := 9 * restCfg.Timeout / 10

	// Preserve any existing WrapTransport configuration by chaining.
	oldWrapTransport := restCfg.WrapTransport
	restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if oldWrapTransport != nil {
			rt = oldWrapTransport(rt)
		}

		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = cfg.retryMax
		retryClient.RetryWaitMin = cfg.retryWaitMin
		retryClient.RetryWaitMax = cfg.retryWaitMax
		retryClient.Logger = cfg.logger
		retryClient.HTTPClient.Timeout = clientTimeout
		retryClient.HTTPClient.Transport = rt
		return retryClient.StandardClient().Transport
	}
}
