package datastore

const (
	// Explain Analyze indicated that 2 statements for PLOP is faster than one.
	deleteOrphanedPLOPDeploymentsAndPI = `DELETE FROM listening_endpoints WHERE processindicatorid in (SELECT id from process_indicators pi WHERE NOT EXISTS
                (SELECT 1 FROM deployments WHERE pi.deploymentid = deployments.Id) AND
                (signal_time < now() at time zone 'utc' - INTERVAL '%d MINUTES' OR signal_time is NULL))`

	deleteOrphanedPLOPPods = `DELETE FROM listening_endpoints WHERE processindicatorid in (SELECT id from process_indicators pi WHERE NOT EXISTS
                (SELECT 1 FROM pods WHERE pi.poduid = pods.Id) AND
                (signal_time < now() at time zone 'utc' - INTERVAL '%d MINUTES' OR signal_time is NULL))`

	// Unfortunately if a listening endpoint is marked as being open there is no indication of how old it is.
	// This leads to a possible race condition where a listening endpoint reaches the database before the deployment,
	// and the pruning job happens to run before the deployment information arrives in the database.
	// This should be rare, so this should be acceptable. This could be improved by adding a timestamp to the listening endpoints table
	deleteOrphanedPLOPDeployments = `DELETE FROM listening_endpoints WHERE NOT EXISTS
                (SELECT 1 FROM deployments WHERE listening_endpoints.deploymentid = deployments.Id)`

	// Unfortunately if a listening endpoint is marked as being open there is no indication of how old it is.
	// This leads to a possible race condition where a listening endpoint reaches the database before the pod,
	// and the pruning job happens to run before the pod information arrives in the database.
	// This should be rare, so this should be acceptable. This could be improved by adding a timestamp to the listening endpoints table
	deleteOrphanedPLOPPodsWithPodUID = `DELETE FROM listening_endpoints WHERE poduid IS NOT NULL AND NOT EXISTS
                (SELECT 1 FROM pods WHERE listening_endpoints.poduid = pods.Id)`

	// Delete orphaned PLOPs
	pruneOrphanedPLOPs = `DELETE FROM listening_endpoints WHERE closetimestamp < now() at time zone 'utc' - INTERVAL '%d MINUTES'`

	// Finds PLOPs without matching process indicators. Not all of these PLOPs are orphaned. There is a further check to see
	// if the serialized data has process information. PLOPs without process information are then deleted.
	getPotentiallyOrphanedPLOPs = `SELECT plop.serialized FROM listening_endpoints plop where NOT EXISTS
			(select 1 FROM process_indicators proc where plop.processindicatorid = proc.id)`
)
