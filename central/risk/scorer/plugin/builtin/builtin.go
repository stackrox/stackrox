package builtin

import (
	"context"

	"github.com/stackrox/rox/central/risk/multipliers/deployment"
	"github.com/stackrox/rox/central/risk/scorer/plugin"
	"github.com/stackrox/rox/generated/storage"
)

// Plugin names for built-in plugins
const (
	PolicyViolationsName    = "policy-violations"
	ProcessBaselinesName    = "process-baselines"
	VulnerabilitiesName     = "vulnerabilities"
	ServiceConfigName       = "service-config"
	PortExposureName        = "port-exposure"
	RiskyComponentsName     = "risky-components"
	ComponentCountName      = "component-count"
	ImageAgeName            = "image-age"
)

// MultiplierPlugin wraps an existing deployment.Multiplier to implement plugin.Plugin.
type MultiplierPlugin struct {
	name       string
	multiplier deployment.Multiplier
}

// NewMultiplierPlugin creates a plugin from an existing multiplier.
func NewMultiplierPlugin(name string, multiplier deployment.Multiplier) plugin.Plugin {
	return &MultiplierPlugin{
		name:       name,
		multiplier: multiplier,
	}
}

// Score delegates to the underlying multiplier.
func (p *MultiplierPlugin) Score(ctx context.Context, dep *storage.Deployment,
	imageRiskResults map[string][]*storage.Risk_Result) *storage.Risk_Result {
	return p.multiplier.Score(ctx, dep, imageRiskResults)
}

// Name returns the plugin identifier.
func (p *MultiplierPlugin) Name() string {
	return p.name
}
