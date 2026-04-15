package plugin

import (
	"context"

	"github.com/stackrox/rox/generated/storage"
)

// Plugin calculates a risk score contribution for a deployment.
// Each plugin evaluates one dimension of risk (e.g., vulnerabilities, policy violations).
type Plugin interface {
	// Score evaluates the deployment and returns a risk result.
	// imageRiskResults maps risk factor names to aggregated results from images.
	// Returns nil if the plugin has nothing to contribute.
	Score(ctx context.Context, deployment *storage.Deployment,
		imageRiskResults map[string][]*storage.Risk_Result) *storage.Risk_Result

	// Name returns the plugin's identifier (e.g., "policy-violations").
	Name() string
}

// PluginType specifies the execution model for a risk scoring plugin.
type PluginType int

const (
	// PluginTypeUnspecified is the zero value.
	PluginTypeUnspecified PluginType = iota
	// PluginTypeBuiltin represents compiled Go plugins.
	PluginTypeBuiltin
	// Future: PluginTypeHTTP, PluginTypeGRPC, PluginTypeWASM, PluginTypeCEL
)

// Config holds runtime configuration for a plugin instance.
type Config struct {
	// ID is the unique identifier for this plugin configuration.
	ID string

	// Name is the human-readable name for the plugin.
	Name string

	// Type specifies how the plugin executes (builtin, http, grpc, etc.).
	Type PluginType

	// Enabled controls whether the plugin contributes to scoring.
	Enabled bool

	// Weight is a multiplier applied to the plugin's raw score.
	Weight float32

	// Priority determines execution order (lower = earlier).
	Priority int32

	// Parameters are plugin-specific configuration values.
	Parameters map[string]string
}

// ConfiguredPlugin pairs a Plugin implementation with its Config.
type ConfiguredPlugin struct {
	Plugin Plugin
	Config *Config
}
