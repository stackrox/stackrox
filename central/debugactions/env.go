package debugactions

import "github.com/stackrox/rox/pkg/env"

var (
	DebugActions = env.RegisterBooleanSetting("ROX_DEBUG_ACTIONS", false)
)
