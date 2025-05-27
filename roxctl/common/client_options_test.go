package common

import (
	"testing"
	"time"

	"github.com/stackrox/rox/roxctl/common/auth"
	roxctlio "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/logger"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
)

// TestFunctionalOptions verifies that each option
// modifies the expected property of the config.
func TestFunctionalOptions(t *testing.T) {
	cfg := &HttpClientConfig{}

	assert.Zero(t, cfg.AuthMethod)
	WithAuthMethod(auth.Anonymous())(cfg)
	assert.NotZero(t, cfg.AuthMethod)

	assert.Zero(t, cfg.RetryExponentialBackoff)
	WithRetryExponentialBackoff(true)(cfg)
	assert.NotZero(t, cfg.RetryExponentialBackoff)

	assert.Zero(t, cfg.ForceHTTP1)
	WithForceHTTP1(true)(cfg)
	assert.NotZero(t, cfg.ForceHTTP1)

	assert.Zero(t, cfg.Logger)
	WithLogger(logger.NewLogger(roxctlio.DefaultIO(), printer.DefaultColorPrinter()))(cfg)
	assert.NotZero(t, cfg.Logger)

	assert.Zero(t, cfg.RetryCount)
	WithRetryCount(1)(cfg)
	assert.NotZero(t, cfg.RetryCount)

	assert.Zero(t, cfg.RetryDelay)
	WithRetryDelay(1 * time.Second)(cfg)
	assert.NotZero(t, cfg.RetryDelay)

	assert.Zero(t, cfg.Timeout)
	WithTimeout(1 * time.Second)(cfg)
	assert.NotZero(t, cfg.Timeout)

	assert.Zero(t, cfg.UseInsecure)
	WithUseInsecure(true)(cfg)
	assert.NotZero(t, cfg.UseInsecure)
}
