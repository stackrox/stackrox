package retryablehttp

import (
	"context"
	"io"
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
	logger                retryablehttp.Logger
	retryMax              int
	retryWaitMax          time.Duration
	retryWaitMin          time.Duration
	responseHeaderTimeout time.Duration
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

// WithResponseHeaderTimeout sets how long to wait for the server to begin
// responding (send response headers) on each attempt. Once headers arrive
// the timeout stops — slow body transfers are not interrupted. When unset,
// each attempt is bounded only by the overall context deadline from
// restCfg.Timeout.
func WithResponseHeaderTimeout(d time.Duration) Option {
	return func(c *config) {
		c.responseHeaderTimeout = d
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

	// Preserve any existing WrapTransport configuration by chaining.
	oldWrapTransport := restCfg.WrapTransport
	restCfg.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
		if oldWrapTransport != nil {
			rt = oldWrapTransport(rt)
		}

		if cfg.responseHeaderTimeout > 0 {
			rt = &headerTimeoutTransport{base: rt, timeout: cfg.responseHeaderTimeout}
		}

		retryClient := retryablehttp.NewClient()
		retryClient.RetryMax = cfg.retryMax
		retryClient.RetryWaitMin = cfg.retryWaitMin
		retryClient.RetryWaitMax = cfg.retryWaitMax
		retryClient.Logger = cfg.logger
		retryClient.HTTPClient.Transport = rt
		return retryClient.StandardClient().Transport
	}
}

// headerTimeoutTransport wraps a RoundTripper to cancel requests when the
// server does not begin responding within the configured timeout. Once
// response headers are received the timer stops — body reads are not
// affected, so slow but active transfers complete normally.
type headerTimeoutTransport struct {
	base    http.RoundTripper
	timeout time.Duration
}

func (t *headerTimeoutTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancel(req.Context())
	timer := time.AfterFunc(t.timeout, cancel)

	resp, err := t.base.RoundTrip(req.WithContext(ctx))
	timer.Stop()
	if err != nil {
		cancel()
		return nil, err
	}

	resp.Body = &cancelOnClose{ReadCloser: resp.Body, cancel: cancel}
	return resp, nil
}

// cancelOnClose calls cancel when the response body is closed, ensuring
// the derived context from headerTimeoutTransport is cleaned up.
type cancelOnClose struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (c *cancelOnClose) Close() error {
	defer c.cancel()
	return c.ReadCloser.Close()
}
