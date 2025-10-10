package manager

import (
	"time"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
)

// EnrichmentResult is the general result of the enrichment process.
type EnrichmentResult string

const (
	// EnrichmentResultRetryLater is returned when the enrichment cannot be done in the current tick and should be retried in the next tick.
	EnrichmentResultRetryLater EnrichmentResult = "retry-later"

	// EnrichmentResultInvalidInput is returned when the given object is broken and misses data required for enriching.
	EnrichmentResultInvalidInput EnrichmentResult = "invalid-input"

	// EnrichmentResultSuccess is returned when the enrichment was successful.
	EnrichmentResultSuccess EnrichmentResult = "success"

	// EnrichmentResultSkipped is returned when the enrichment was skipped. There is no need to retry.
	// The input to the enrichment can be discarded as it brings no new information.
	EnrichmentResultSkipped EnrichmentResult = "skipped"

	// EnrichmentResultContainerIDMissMarkRotten is returned when the enrichment cannot be done because the containerID is not found.
	// The item is marked as rotten and will be removed from the enrichment queue after the enrichment pipeline is done.
	// Connection or endpoint is marked as rotten when the containerID is unknown and the entity was not found in activeConnections or activeEndpoints.
	// This usually means that the container was deleted after Collector reported the closing of the connection/endpoint.
	EnrichmentResultContainerIDMissMarkRotten EnrichmentResult = "container-id-not-found-mark-rotten"

	// EnrichmentResultContainerIDMissMarkInactive is returned when the enrichment cannot be done because the containerID is not found.
	// The item is marked as inactive and will be removed from the enrichment queue after the enrichment pipeline is done.
	// Connection or endpoint is marked as inactive when the containerID is unknown but the entity was found in activeConnections or activeEndpoints.
	// This usually means that the container was deleted before Collector reported the closing of the connection/endpoint.
	EnrichmentResultContainerIDMissMarkInactive EnrichmentResult = "container-id-not-found-mark-inactive"
)

// PostEnrichmentAction is the action to be executed after the enrichment is done.
type PostEnrichmentAction string

const (
	// PostEnrichmentActionRemove is returned when the enrichment is done successfully and the item should be removed from the enrichment queue.
	PostEnrichmentActionRemove PostEnrichmentAction = "remove"

	// PostEnrichmentActionRetry is returned when the enrichment is missing information required for enrichment, but the information may arrive later.
	// The item should be retried in the next tick.
	PostEnrichmentActionRetry PostEnrichmentAction = "retry"

	// PostEnrichmentActionMarkInactive is returned when the enrichment is done successfully and the item should be marked as inactive.
	// It means that the item was active, but based on the information from Collector or Sensor, it should be deactivated (i.e., connection or endpoint was closed).
	PostEnrichmentActionMarkInactive PostEnrichmentAction = "mark-inactive"

	// PostEnrichmentActionCheckRemove is returned when the item should be checked for removal from the enrichment queue.
	// This is used when the enrichment is finished, but we must check additional information to decide if the item should be removed.
	// For example, if the endpoint enrichment is finished successfully, but we still may need to keep the data to do the PLoP enrichment.
	PostEnrichmentActionCheckRemove PostEnrichmentAction = "check-remove"
)

// EnrichmentReasonConn provides additional information about given EnrichmentResult for Connections.
// It should explain why the given EnrichmentResult was returned.
type EnrichmentReasonConn string

const (
	// EnrichmentReasonConnSuccess is returned when the enrichment was successful and we don't need to provide more information.
	EnrichmentReasonConnSuccess EnrichmentReasonConn = "success"

	// EnrichmentReasonConnStillInGracePeriod is returned when the enrichment was not done because the containerID is not found and we are still within the grace period.
	// This happens when Collector message arrived before the containerID was resolved by k8s informers.
	EnrichmentReasonConnStillInGracePeriod EnrichmentReasonConn = "still-in-grace-period"

	// EnrichmentReasonConnOutsideOfGracePeriod is returned when the enrichment was not done because the containerID is not found and we are past the grace period.
	// This happens when Collector message arrived at a point in time when the k8s informers should have already resolved the containerID, but for some reason did not.
	// This can happen, for example, when the container is deleted and the history feature of clusterEntitiesStore is disabled.
	EnrichmentReasonConnOutsideOfGracePeriod EnrichmentReasonConn = "outside-of-grace-period"

	// EnrichmentReasonConnIPUnresolvableYet is returned when the enrichment was not done because the IP address is valid but not resolvable yet.
	// It means that we are still within the grace period, but we cannot find any deployment for the IP address.
	// When the grace period is over and we still cannot find any deployment or known CIDR for the IP address,
	// the entity is represented as "External-" or "Internal-Entities" on the network graph.
	EnrichmentReasonConnIPUnresolvableYet EnrichmentReasonConn = "ip-unresolvable-yet"

	// EnrichmentReasonConnLookupByNetworkFailed is returned when the enrichment was not done
	// because the IP address is not valid and we suspect that only the network part of the IP address is provided by Collector.
	// In this case, we attempted to find a matching network CIDR for the network part of the IP address but that failed.
	// As this is only returned while being still within the grace period, we will retry the enrichment later.
	EnrichmentReasonConnLookupByNetworkFailed EnrichmentReasonConn = "lookup-by-network-failed"

	// EnrichmentReasonConnParsingIPFailed is returned when the enrichment was not done because the IP address is malformed.
	// This is connected with a bug in Collector (ROX-22388) and should not happen after the bug is fixed.
	// In any case, when IP is malformed, we return `EnrichmentResultInvalidInput` and do not retry the enrichment again.
	EnrichmentReasonConnParsingIPFailed EnrichmentReasonConn = "parsing-ip-failed"

	// EnrichmentReasonConnIncomingInternalConnection is returned when the enrichment was not done because the connection is incoming and local.
	// This happens when both ends of a connection are in the same cluster and Sensor will get two updates about the same connection.
	// We arbitrary decide to skip enriching one of them - the incoming one. The enrichment will still be done for the outgoing part of the connection.
	// This is done to avoid duplicate entries in the network graph.
	EnrichmentReasonConnIncomingInternalConnection EnrichmentReasonConn = "incoming-internal-connection"
)

// EnrichmentReasonEp provides additional information about given EnrichmentResult for endpoints.
// It should explain why the given EnrichmentResult was returned.
type EnrichmentReasonEp string

const (
	// EnrichmentReasonEpStillInGracePeriod is returned when the enrichment was not done because the containerID is not found and we are still within the grace period.
	// This happens when Collector message arrived before the containerID was resolved by k8s informers.
	EnrichmentReasonEpStillInGracePeriod EnrichmentReasonEp = "still-in-grace-period"

	// EnrichmentReasonEpOutsideOfGracePeriod is returned when the enrichment was not done because the containerID is not found and we are past the grace period.
	// This happens when Collector message arrived at a point in time when the k8s informers should have already resolved the containerID, but for some reason did not.
	// This can happen, for example, when the container is deleted and the history feature of clusterEntitiesStore is disabled.
	EnrichmentReasonEpOutsideOfGracePeriod EnrichmentReasonEp = "outside-of-grace-period"

	// EnrichmentReasonEpEmptyProcessInfo is returned when PLoP (Processes Listening on Ports) enrichment cannot proceed because process information is missing or empty.
	// This means the endpoint was detected but without sufficient process details to enrich the "processes listening on ports" data.
	EnrichmentReasonEpEmptyProcessInfo EnrichmentReasonEp = "empty-process-info"

	// EnrichmentReasonEpDuplicate is returned when the enrichment was skipped because we already have more recent data for this endpoint.
	// Multiple collector messages can report the same endpoint, so we only process the most recent one.
	EnrichmentReasonEpDuplicate EnrichmentReasonEp = "duplicate"

	// EnrichmentReasonEpFeaturePlopDisabled is returned when the ProcessesListeningOnPort feature is disabled.
	// This means PLoP (Processes Listening on Ports) enrichment will not be performed.
	EnrichmentReasonEpFeaturePlopDisabled EnrichmentReasonEp = "feature-plop-disabled"

	// EnrichmentReasonEpSuccessActive is returned when the enrichment was successful for an active (open) endpoint.
	// The endpoint is currently listening and has been added to the active endpoints tracking.
	EnrichmentReasonEpSuccessActive EnrichmentReasonEp = "success-active"

	// EnrichmentReasonEpSuccessInactive is returned when the enrichment was successful for an inactive (closed) endpoint.
	// The endpoint was previously listening but has since been closed.
	EnrichmentReasonEpSuccessInactive EnrichmentReasonEp = "success-inactive"
)

// enrichmentConsumption is a helper struct to track if the enrichment was consumed by the network graph and PLOP.
// It is used to make sure that both enrichment pipelines are executed on the same item.
type enrichmentConsumption struct {
	consumedNetworkGraph bool
	consumedPLOP         bool
}

// IsConsumed checks that network graph (and PLOP if enabled) used the enrichment result.
func (e *enrichmentConsumption) IsConsumed() bool {
	if env.ProcessesListeningOnPort.BooleanSetting() {
		return e.consumedNetworkGraph && e.consumedPLOP
	}
	return e.consumedNetworkGraph
}

func (c *connStatus) isClosed() bool {
	return c.lastSeen != timestamp.InfiniteFuture
}

type connStatus struct {
	// Order of fields is optimized for memory alignment.
	// See https://goperf.dev/01-common-patterns/fields-alignment/ for more details.

	// tsAdded is a timestamp when this item was added to Sensor's memory (in connectionsByHost)
	tsAdded timestamp.MicroTS
	// firstSeen is a timestamp of opening a given connection/endpoint
	firstSeen timestamp.MicroTS
	// lastSeen is a timestamp of closing a given connection/endpoint.
	// It is set to infinity for items that are still open.
	lastSeen timestamp.MicroTS

	enrichmentConsumption enrichmentConsumption

	// isExternal
	isExternal bool
	// rotten means that the item does not map to any known containerID and the grace-period has passed
	rotten bool
	// historicalContainerID means that the item maps to a containerID that has recently been deleted
	historicalContainerID bool
	// containerIDFound is set when the item can be mapped to a known containerID (historical or not)
	containerIDFound bool
}

func (c *connStatus) timeElapsedSinceFirstSeen(now timestamp.MicroTS) time.Duration {
	return now.ElapsedSince(c.firstSeen)
}
func (c *connStatus) isFresh(now timestamp.MicroTS) bool {
	return c.timeElapsedSinceFirstSeen(now) < env.ClusterEntityResolutionWaitPeriod.DurationSetting()
}
func (c *connStatus) pastContainerResolutionDeadline(now timestamp.MicroTS) bool {
	return c.timeElapsedSinceFirstSeen(now) > env.ContainerIDResolutionGracePeriod.DurationSetting()
}

// checkRemoveCondition returns true when the given entity can be removed from the enrichment queue.
// It returns false if the enrichment should be retried in the next tick.
func (c *connStatus) checkRemoveCondition(useLegacy, isConsumed bool) bool {
	// Legacy UpdateComputer requires keeping all open entities in the enrichment queue until they are closed.
	if useLegacy {
		return c.rotten || (c.isClosed() && isConsumed)
	}
	return c.rotten || isConsumed
}

// containerResolutionResult holds the result of container ID resolution
type containerResolutionResult struct {
	Container          clusterentities.ContainerMetadata
	Found              bool
	IsHistorical       bool
	ShouldRetryLater   bool
	DeactivationResult EnrichmentResult
}

// ActiveEntity defines the union of types that can be used for container ID resolution
type ActiveEntity interface {
	connection | containerEndpoint
}

// ActiveEntityChecker is a generic interface for checking if an entity exists in active collections
type ActiveEntityChecker[T ActiveEntity] interface {
	IsActive(entity T) bool
}

// endpointActiveChecker implements ActiveEntityChecker for container endpoints
type endpointActiveChecker struct {
	mutex           *sync.RWMutex
	activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge
}

func (c *endpointActiveChecker) IsActive(ep containerEndpoint) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, found := c.activeEndpoints[ep]
	return found
}

// connectionActiveChecker implements ActiveEntityChecker for connections
type connectionActiveChecker struct {
	mutex             *sync.RWMutex
	activeConnections map[connection]*networkConnIndicatorWithAge
}

func (c *connectionActiveChecker) IsActive(conn connection) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	_, found := c.activeConnections[conn]
	return found
}

// resolveContainerID performs the shared container ID resolution logic using generics
func resolveContainerID[T ActiveEntity](
	m *networkFlowManager,
	now timestamp.MicroTS,
	containerID string,
	status *connStatus,
	activeChecker ActiveEntityChecker[T],
	entity T,
) containerResolutionResult {
	// Look up container metadata
	container, contIDfound, isHistorical := m.clusterEntities.LookupByContainerID(containerID)

	// Update status with lookup results
	status.historicalContainerID = isHistorical
	status.containerIDFound = contIDfound

	result := containerResolutionResult{
		Container:    container,
		Found:        contIDfound,
		IsHistorical: isHistorical,
	}

	// Handle case where container ID is not found
	if !contIDfound {
		// Check if we're still within grace period
		if !status.pastContainerResolutionDeadline(now) {
			result.ShouldRetryLater = true
			return result
		}

		// Past grace period - determine deactivation result based on active status
		if activeChecker.IsActive(entity) {
			result.DeactivationResult = EnrichmentResultContainerIDMissMarkInactive
		} else {
			result.DeactivationResult = EnrichmentResultContainerIDMissMarkRotten
		}
	}

	return result
}
