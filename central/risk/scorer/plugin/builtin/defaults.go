package builtin

import (
	"github.com/stackrox/rox/central/processbaseline/evaluator"
	"github.com/stackrox/rox/central/risk/getters"
	"github.com/stackrox/rox/central/risk/multipliers/deployment"
	"github.com/stackrox/rox/central/risk/multipliers/image"
	"github.com/stackrox/rox/central/risk/scorer/plugin"
	"github.com/stackrox/rox/central/risk/scorer/plugin/registry"
)

// DefaultConfig represents the default configuration for a built-in plugin.
type DefaultConfig struct {
	ID       string
	Name     string
	Priority int32
	Weight   float32
}

// DefaultConfigs returns the default configuration for all built-in plugins.
// Order matches the existing multiplier execution order.
func DefaultConfigs() []DefaultConfig {
	return []DefaultConfig{
		{ID: "builtin-policy-violations", Name: PolicyViolationsName, Priority: 100, Weight: 1.0},
		{ID: "builtin-process-baselines", Name: ProcessBaselinesName, Priority: 200, Weight: 1.0},
		{ID: "builtin-vulnerabilities", Name: VulnerabilitiesName, Priority: 300, Weight: 1.0},
		{ID: "builtin-service-config", Name: ServiceConfigName, Priority: 400, Weight: 1.0},
		{ID: "builtin-port-exposure", Name: PortExposureName, Priority: 500, Weight: 1.0},
		{ID: "builtin-risky-components", Name: RiskyComponentsName, Priority: 600, Weight: 1.0},
		{ID: "builtin-component-count", Name: ComponentCountName, Priority: 700, Weight: 1.0},
		{ID: "builtin-image-age", Name: ImageAgeName, Priority: 800, Weight: 1.0},
	}
}

// RegisterPlugins registers all built-in plugins with the registry.
// This should be called during Central initialization.
func RegisterPlugins(reg registry.Registry, alertSearcher getters.AlertSearcher, baselineEvaluator evaluator.Evaluator) {
	// Register policy violations plugin
	reg.Register(NewMultiplierPlugin(
		PolicyViolationsName,
		deployment.NewViolations(alertSearcher),
	))

	// Register process baselines plugin
	reg.Register(NewMultiplierPlugin(
		ProcessBaselinesName,
		deployment.NewProcessBaselines(baselineEvaluator),
	))

	// Register vulnerabilities plugin (via image risk aggregation)
	reg.Register(NewMultiplierPlugin(
		VulnerabilitiesName,
		deployment.NewImageMultiplier(image.VulnerabilitiesHeading),
	))

	// Register service config plugin
	reg.Register(NewMultiplierPlugin(
		ServiceConfigName,
		deployment.NewServiceConfig(),
	))

	// Register port exposure plugin
	reg.Register(NewMultiplierPlugin(
		PortExposureName,
		deployment.NewReachability(),
	))

	// Register risky components plugin (via image risk aggregation)
	reg.Register(NewMultiplierPlugin(
		RiskyComponentsName,
		deployment.NewImageMultiplier(image.RiskyComponentCountHeading),
	))

	// Register component count plugin (via image risk aggregation)
	reg.Register(NewMultiplierPlugin(
		ComponentCountName,
		deployment.NewImageMultiplier(image.ComponentCountHeading),
	))

	// Register image age plugin (via image risk aggregation)
	reg.Register(NewMultiplierPlugin(
		ImageAgeName,
		deployment.NewImageMultiplier(image.ImageAgeHeading),
	))
}

// SetupDefaultConfigs creates default plugin configurations in the registry.
// These enable the plugins with default weights and priorities.
func SetupDefaultConfigs(reg registry.Registry) {
	for _, dc := range DefaultConfigs() {
		config := &plugin.Config{
			ID:       dc.ID,
			Name:     dc.Name,
			Type:     plugin.PluginTypeBuiltin,
			Enabled:  true,
			Weight:   dc.Weight,
			Priority: dc.Priority,
		}
		_ = reg.UpsertConfig(config)
	}
}
