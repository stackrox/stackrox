package complianceoperator

import "github.com/stackrox/rox/pkg/env"

var (
	syncScanConfigsOnStartup = env.RegisterBooleanSetting("ROX_SYNC_SCAN_CONFIGS_ON_STARTUP", true)
)
