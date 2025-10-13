package types

import (
	gcpAuth "github.com/stackrox/rox/pkg/cloudproviders/gcp/auth"
)

// CreatorConfig specifies optional configuration parameters for registry creators.
type CreatorConfig struct {
	GCPTokenManager gcpAuth.STSTokenManager
	MetricsHandler  *MetricsHandler
}

// GetGCPTokenManager is a nil-safe getter for GCPTokenManager.
func (c *CreatorConfig) GetGCPTokenManager() gcpAuth.STSTokenManager {
	if c == nil {
		return nil
	}
	return c.GCPTokenManager
}

// GetMetricsHandler is a nil-safe getter for MetricsHandler.
func (c *CreatorConfig) GetMetricsHandler() *MetricsHandler {
	if c == nil {
		return nil
	}
	return c.MetricsHandler
}

// CreatorOption is a functor that applies the creator config option.
type CreatorOption func(opt *CreatorConfig) *CreatorConfig

// WithGCPTokenManager adds a GCP token manager.
func WithGCPTokenManager(manager gcpAuth.STSTokenManager) CreatorOption {
	return func(opt *CreatorConfig) *CreatorConfig {
		if opt == nil {
			opt = &CreatorConfig{}
		}
		opt.GCPTokenManager = manager
		return opt
	}
}

// WithMetricsHandler adds a Prometheus metrics handler.
func WithMetricsHandler(handler *MetricsHandler) CreatorOption {
	return func(opt *CreatorConfig) *CreatorConfig {
		if opt == nil {
			opt = &CreatorConfig{}
		}
		opt.MetricsHandler = handler
		return opt
	}
}

// ApplyCreatorOptions applies all options and returns the creator config.
func ApplyCreatorOptions(options ...CreatorOption) *CreatorConfig {
	cfg := &CreatorConfig{}
	for _, opt := range options {
		cfg = opt(cfg)
	}
	return cfg
}
