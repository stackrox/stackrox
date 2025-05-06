package common

import (
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/roxctl/common/auth"
	"github.com/stackrox/rox/roxctl/common/flags"
	"github.com/stackrox/rox/roxctl/common/logger"
)

// HttpClientOption encodes behavior of a HTTP client.
type HttpClientOption func(*HttpClientConfig)

// HttpClientConfig is used for configuring the abstracted HTTP client.
type HttpClientConfig struct {
	AuthMethod              auth.Method
	ForceHTTP1              bool
	Logger                  logger.Logger
	RetryExponentialBackoff bool
	RetryCount              int
	RetryDelay              time.Duration
	Timeout                 time.Duration
	UseInsecure             bool
}

// NewHttpClientConfig returns a default config modified by options.
func NewHttpClientConfig(options ...HttpClientOption) *HttpClientConfig {
	opts := &HttpClientConfig{
		ForceHTTP1:              flags.ForceHTTP1(),
		UseInsecure:             flags.UseInsecure(),
		RetryCount:              env.ClientMaxRetries.IntegerSetting(),
		RetryDelay:              10 * time.Second,
		RetryExponentialBackoff: true,
	}

	for _, optFunc := range options {
		optFunc(opts)
	}

	return opts
}

// WithAuthMethod sets the auth method to use for the HTTP client.
func WithAuthMethod(am auth.Method) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.AuthMethod = am
	}
}

// WithRetryExponentialBackoff disables/enables exponential backoff.
func WithRetryExponentialBackoff(value bool) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.RetryExponentialBackoff = value
	}
}

// WithForceHTTP1 sets if the client should only use HTTP1.
func WithForceHTTP1(force bool) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.ForceHTTP1 = force
	}
}

// WithLogger sets the logger the client should use.
func WithLogger(log logger.Logger) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.Logger = log
	}
}

// WithRetryCount sets the number of retry attempts on request failure.
func WithRetryCount(retryCount int) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.RetryCount = retryCount
	}
}

// WithRetryDelay sets the time to wait between retry attempts.
func WithRetryDelay(d time.Duration) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.RetryDelay = d
	}
}

// WithTimeout the timeout to use for the http request.
func WithTimeout(timeout time.Duration) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.Timeout = timeout
	}
}

// WithUseInsecure sets if HTTP1 should be forced.
func WithUseInsecure(useInsecure bool) HttpClientOption {
	return func(hco *HttpClientConfig) {
		hco.UseInsecure = useInsecure
	}
}
