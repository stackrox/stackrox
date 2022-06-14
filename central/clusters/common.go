package clusters

import "github.com/stackrox/rox/pkg/env"

var (
	// CollectorModuleDownloadBaseURL is the canonical upstream location for collector modules.
	CollectorModuleDownloadBaseURL = env.RegisterSetting(
		`ROX_COLLECTOR_MODULES_BASE_URL`,
		env.WithDefault("https://collector-modules.stackrox.io/612dd2ee06b660e728292de9393e18c81a88f347ec52a39207c5166b5302b656"),
	)
)
