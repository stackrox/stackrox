package reconciler

import "github.com/stackrox/rox/pkg/env"

var (
	collectorRegistryOverride = env.RegisterSetting("ROX_OPERATOR_MAIN_REGISTRY", env.WithDefault("quay.io/stackrox-io"))
	mainRegistryOverride      = env.RegisterSetting("ROX_OPERATOR_COLLECTOR_REGISTRY", env.WithDefault("quay.io/stackrox-io"))
)
