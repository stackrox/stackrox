package resources

import "github.com/stackrox/rox/pkg/env"

var pastEndpointsMemorySize = env.RegisterSetting("ROX_PAST_ENDPOINTS_MEMORY_SIZE", env.WithDefault("0"))
