package manager

import (
	"strconv"
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timestamp"
)

const (
	// Wait at least this long before determining that an unresolvable IP is "outside of the cluster".
	clusterEntityResolutionWaitPeriod = 10 * time.Second
)

type EnrichmentResult string

const (
	EnrichmentResultRetryLater EnrichmentResult = "retry-later"
	// EnrichmentResultInvalidInput is returned when the given object is broken and misses data required for enriching
	EnrichmentResultInvalidInput EnrichmentResult = "invalid-input"

	EnrichmentResultSuccess EnrichmentResult = "success"
	EnrichmentResultSkipped EnrichmentResult = "skipped"

	EnrichmentResultContainerIDMissMarkRotten   EnrichmentResult = "container-id-not-found-mark-rotten"
	EnrichmentResultContainerIDMissMarkInactive EnrichmentResult = "container-id-not-found-mark-inactive"
)

type PostEnrichmentAction string

const (
	PostEnrichmentActionRemove       PostEnrichmentAction = "remove"
	PostEnrichmentActionRetry        PostEnrichmentAction = "retry"
	PostEnrichmentActionMarkInactive PostEnrichmentAction = "mark-inactive"
	PostEnrichmentActionCheckRemove  PostEnrichmentAction = "check-remove"
)

type enrichmentResult struct {
	consumedNetworkGraph bool
	consumedPLOP         bool
}

// IsConsumed checks that network graph (and PLOP if enabled) used the enrichment result.
func (e *enrichmentResult) IsConsumed() bool {
	if env.ProcessesListeningOnPort.BooleanSetting() {
		return e.consumedNetworkGraph && e.consumedPLOP
	}
	return e.consumedNetworkGraph
}

func (c *connStatus) isClosed() bool {
	return c.lastSeen != timestamp.InfiniteFuture
}

type connStatus struct {
	// tsAdded is a timestamp when this item was added to Sensor's memory (in connectionsByHost)
	tsAdded timestamp.MicroTS
	// firstSeen is a timestamp of opening a given connection/endpoint
	firstSeen timestamp.MicroTS
	// lastSeen is a timestamp of closing a given connection/endpoint.
	// It is set to infinity for items that are still open.
	lastSeen timestamp.MicroTS
	// rotten means that the item does not map to any known containerID and the grace-period has passed
	rotten bool
	// historicalContainerID means that the item maps to a containerID that has recently been deleted
	historicalContainerID bool
	// containerIDFound is set when the item can be mapped to a known containerID (historical or not)
	containerIDFound bool
	// isExternal
	isExternal string

	enrichmentResult enrichmentResult
}

func (c *connStatus) timeElapsedSinceFirstSeen(now timestamp.MicroTS) time.Duration {
	return now.ElapsedSince(c.firstSeen)
}
func (c *connStatus) isFresh(now timestamp.MicroTS) bool {
	return c.timeElapsedSinceFirstSeen(now) < clusterEntityResolutionWaitPeriod
}
func (c *connStatus) pastContainerResolutionDeadline(now timestamp.MicroTS) bool {
	return c.timeElapsedSinceFirstSeen(now) > env.ContainerIDResolutionGracePeriod.DurationSetting()
}
func (c *connStatus) setIsExternal(isExternal bool) {
	c.isExternal = strconv.FormatBool(isExternal)
}

func logReasonForAggregatingNetGraphFlow(conn *connection, contNs, contName, entitiesName string, port uint16) {
	reasonStr := ""
	// No need to produce complex chain of reasons, if there is one simple explanation
	if conn.remote.IsConsideredExternal() {
		reasonStr = "Collector did not report the IP address to Sensor - the remote part is the Internet"
	}
	if conn.incoming {
		// Keep internal wording even if central lacks `NetworkGraphInternalEntitiesSupported` capability.
		log.Debugf("Marking incoming connection to container %s/%s from %s:%s as '%s' in the network graph: %s.",
			contNs, contName, conn.remote.IPAndPort.String(),
			strconv.Itoa(int(port)), entitiesName, reasonStr)
	} else {
		log.Debugf("Marking outgoing connection from container %s/%s to %s as '%s' in the network graph: %s.",
			contNs, contName, conn.remote.IPAndPort.String(),
			entitiesName, reasonStr)
	}
}
