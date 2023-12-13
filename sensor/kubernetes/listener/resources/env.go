package resources

import "github.com/stackrox/rox/pkg/env"

// pastEndpointsMemorySize defines for how many ticks Sensor should remember past endpoints.
// The duration of one tick is defined as duration of defnetworkFlowManager.enricherTicker.
// Value of 0 disables the memory completely.
// The default value of 20 results in between 9.5 to 10 minutes (19-20 * 30s) - the precision of 1 tick is necessary,
// as we have no influence in which part of the 30s window a deletion event from K8s informer arrives.
// A memory of ~10 minutes is necessary if Collector is used with ROX_ENABLE_AFTERGLOW=true and its default settings.
// This means that Collector can report on past connections up to 400 seconds (for default Collector settings)
// after k8s informs Sensor that an Endpoint (or IP address) has been deleted.
// Setting this value too low may result in 'External Entities' appearing on the Network Graph
// for connections that are in fact not external.
// Setting this value too high will result in higher memory consumption of Sensor (especially in clusters with many
// deletions or dynamic changes to services or IP addresses).
var pastEndpointsMemorySize = env.RegisterIntegerSetting("ROX_PAST_ENDPOINTS_MEMORY_SIZE", 20)
