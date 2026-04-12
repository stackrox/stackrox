package types

import (
	"golang.org/x/oauth2"
)

// TokenManager provides OAuth2 token management for cloud provider authentication.
// Defined here instead of importing cloud-specific packages to avoid dragging
// the entire GCP/AWS/Azure SDK chain into every registry type.
type TokenManager interface {
	Start()
	Stop()
	TokenSource() oauth2.TokenSource
}

// CreatorConfig specifies optional configuration parameters for registry creators.
type CreatorConfig struct {
	GCPTokenManager TokenManager
	MetricsHandler  *MetricsHandler
}

// GetGCPTokenManager is a nil-safe getter for GCPTokenManager.
func (c *CreatorConfig) GetGCPTokenManager() TokenManager {
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
func WithGCPTokenManager(manager TokenManager) CreatorOption {
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
