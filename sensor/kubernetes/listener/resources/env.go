package resources

import "github.com/stackrox/rox/pkg/env"

var (
	// PastClusterEntitiesMemorySize defines for how many ticks Sensor should remember past entities.
	// TODO(ROX-28259): Re-enable ROX_PAST_CLUSTER_ENTITIES_MEMORY_SIZE option.
	// The duration of one tick is defined as duration of networkFlowManager.enricherTicker.
	// Value of 0 disables the memory completely.
	// The default value of 20 results in between 9.5 to 10 minutes (19-20 * 30s) - the precision of 1 tick is necessary,
	// as we have no influence in which part of the 30s window a deletion event from K8s informer arrives.
	// A memory of ~10 minutes is necessary if Collector is used with ROX_ENABLE_AFTERGLOW=true and its default settings.
	// This means that Collector can report on past connections up to 400 seconds (for default Collector settings)
	// after k8s informs Sensor that an Endpoint (or IP address) has been deleted.
	// Setting this value too low may result in External- or Internal Entities appearing on the Network Graph
	// for connections that could have been attributed to a past deployment.
	// Setting this value too high will result in higher memory consumption of Sensor (especially in clusters with many
	// deletions or frequent dynamic changes to services or IP addresses).
	PastClusterEntitiesMemorySize = env.RegisterIntegerSetting("ROX_PAST_CLUSTER_ENTITIES_MEMORY_SIZE", 20)
	// debugClusterEntitiesStore enables running a debug http server that allows to look into the state of the
	// clusterentities store and events that added and deleted entries from the store. DO NOT RUN IN PRODUCTION.
	debugClusterEntitiesStore = env.RegisterBooleanSetting("ROX_DEBUG_CLUSTER_ENTITIES_STORE", false)
	// allowHostNetworkPodIPS (EXPERIMENTAL) enables registering specific pod IPs in the clusterentities store.
	// Those IPs were originally skipped due to the following problem: (ROX-897)
	// "When connecting to a nginx deployment (via curl) from an external source, the outEdge for the INTERNET node does not get generated."
	// Allowing those IPs to be stored should visibly decrease the number of cases where we must show "Internal Entities"
	// on the network graph because the deployment owning a given IP could not be found.
	allowHostNetworkPodIPsInEntitiesStore = env.RegisterBooleanSetting("ROX_ALLOW_HOST_NETWORK_POD_IPS_IN_ENTITIES_STORE", false)
)
