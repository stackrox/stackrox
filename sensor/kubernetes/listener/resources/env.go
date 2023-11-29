package resources

import "github.com/stackrox/rox/pkg/env"

var pastEndpointsMemorySize = env.RegisterIntegerSetting("ROX_PAST_ENDPOINTS_MEMORY_SIZE", 2)
