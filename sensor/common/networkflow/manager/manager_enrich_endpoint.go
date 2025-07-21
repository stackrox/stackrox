package manager

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

// EnrichmentReasonEp provides additional information about given EnrichmentResult for endpoints
type EnrichmentReasonEp string

const (
	EnrichmentReasonEpStillInGracePeriod   EnrichmentReasonEp = "still-in-grace-period"
	EnrichmentReasonEpOutsideOfGracePeriod EnrichmentReasonEp = "outside-of-grace-period"
	EnrichmentReasonEpEmptyProcessInfo     EnrichmentReasonEp = "empty-process-info"
	EnrichmentReasonEpDuplicate            EnrichmentReasonEp = "duplicate"
	EnrichmentReasonEpFeatureDisabled      EnrichmentReasonEp = "feature-disabled"
	EnrichmentReasonEpFeaturePlopDisabled  EnrichmentReasonEp = "feature-plop-disabled"
	EnrichmentReasonEpSuccessActive        EnrichmentReasonEp = "success-active"
	EnrichmentReasonEpSuccessInactive      EnrichmentReasonEp = "success-inactive"
)

func (m *networkFlowManager) enrichHostContainerEndpoints(now timestamp.MicroTS, hostConns *hostConnections, enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS, processesListening map[processListeningIndicator]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	flowMetrics.HostProcessesEvents.WithLabelValues("add").Add(float64(len(hostConns.endpoints)))
	flowMetrics.HostConnectionsOperations.WithLabelValues("enrich", "endpoints").Add(float64(len(hostConns.endpoints)))
	for ep, status := range hostConns.endpoints {
		resultNG, resultPLOP, reasonNG, reasonPLOP := m.enrichContainerEndpoint(now, &ep, status, enrichedEndpoints, processesListening, timestamp.Now())
		action := m.handleEndpointEnrichmentResult(resultNG, resultPLOP, reasonNG, reasonPLOP, &ep)
		updateEndpointMetric(now, action, resultNG, resultPLOP, reasonNG, reasonPLOP, status)
		switch action {
		case PostEnrichmentActionRemove:
			delete(hostConns.endpoints, ep)
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "endpoints").Inc()
		case PostEnrichmentActionMarkInactive:
			concurrency.WithLock(&m.activeEndpointsMutex, func() {
				if ok := deactivateEndpointNoLock(&ep, m.activeEndpoints, enrichedEndpoints, now); !ok {
					log.Debugf("Cannot mark endpoint as inactive: endpoint is rotten")
				}
			})
		case PostEnrichmentActionRetry:
		// noop, retry happens through not removing from `hostConns.endpoints`
		case PostEnrichmentActionCheckRemove:
			if status.isClosed() {
				delete(hostConns.endpoints, ep)
				flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "endpoints").Inc()
				flowMetrics.HostProcessesEvents.WithLabelValues("remove").Inc()
			}
		default:
			log.Warnf("Unknown enrichment action: %v", action)
		}
	}
	concurrency.WithLock(&m.activeEndpointsMutex, func() {
		flowMetrics.SetActiveEndpointsTotalGauge(len(m.activeEndpoints))
	})
}

// enrichContainerEndpoint updates `enrichedEndpoints` and `m.activeEndpoints`.
// It returns the enrichment result and provides reason for returning such result.
// Additionally, it sets the outcome in the `status` field to reflect the outcome of the enrichment
// in memory-efficient way by avoiding copying.
func (m *networkFlowManager) enrichContainerEndpoint(
	now timestamp.MicroTS,
	ep *containerEndpoint,
	status *connStatus,
	enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS,
	processesListening map[processListeningIndicator]timestamp.MicroTS,
	lastUpdate timestamp.MicroTS,
) (resultNG, resultPLOP EnrichmentResult, reasonNG, reasonPLOP EnrichmentReasonEp) {
	timeElapsedSinceFirstSeen := now.ElapsedSince(status.firstSeen)
	pastContainerResolutionDeadline := timeElapsedSinceFirstSeen > env.ContainerIDResolutionGracePeriod.DurationSetting()
	isFresh := timeElapsedSinceFirstSeen < clusterEntityResolutionWaitPeriod
	if !isFresh {
		status.enrichmentResult.consumedNetworkGraph = true
	}

	container, contIDfound, isHistorical := m.clusterEntities.LookupByContainerID(ep.containerID)
	status.historicalContainerID = isHistorical
	status.containerIDFound = contIDfound
	if !contIDfound {
		// There is an endpoint involving a container that Sensor does not recognize. In this case we may do two things:
		// (1) decide that we want to retry the enrichment later (keep the endpoint in hostConnections)
		// - this is done while still within the containerID resolution grace period,
		// (2) remove the endpoint from hostConnections, because enrichment is impossible
		// - this is done after the containerID resolution grace period.
		if !pastContainerResolutionDeadline {
			return EnrichmentResultRetryLater, EnrichmentResultRetryLater,
				EnrichmentReasonEpStillInGracePeriod, EnrichmentReasonEpStillInGracePeriod
		}
		// Expire the connection if we are past the containerID resolution grace period.
		result := concurrency.WithLock1(&m.activeEndpointsMutex, func() EnrichmentResult {
			if _, found := m.activeEndpoints[*ep]; !found {
				return EnrichmentResultContainerIDMissMarkRotten
			}
			return EnrichmentResultContainerIDMissMarkInactive
		})
		return result, result, EnrichmentReasonEpOutsideOfGracePeriod, EnrichmentReasonEpOutsideOfGracePeriod
	}

	// SECTION: ENRICHMENT OF PROCESSES LISTENING ON PORTS
	if env.ProcessesListeningOnPort.BooleanSetting() {
		status.enrichmentResult.consumedPLOP = true
		resultPLOP, reasonPLOP = m.enrichPLOP(ep, container, processesListening, status.lastSeen)
	} else {
		resultPLOP = EnrichmentResultSkipped
		reasonPLOP = EnrichmentReasonEpFeaturePlopDisabled
	}

	// SECTION: ENRICHMENT OF ENDPOINT
	status.enrichmentResult.consumedNetworkGraph = true
	indicator := containerEndpointIndicator{
		entity:   networkgraph.EntityForDeployment(container.DeploymentID),
		port:     ep.endpoint.IPAndPort.Port,
		protocol: ep.endpoint.L4Proto.ToProtobuf(),
	}

	// Multiple endpoints from a collector can result in a single enriched endpoint,
	// hence update the timestamp only if we have a more recent endpoint than the one we have already enriched.
	if oldTS, found := enrichedEndpoints[indicator]; found && oldTS >= status.lastSeen {
		return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpDuplicate, reasonPLOP
	}

	enrichedEndpoints[indicator] = status.lastSeen
	if !features.SensorCapturesIntermediateEvents.Enabled() {
		return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpFeatureDisabled, reasonPLOP
	}

	m.activeEndpointsMutex.Lock()
	defer m.activeEndpointsMutex.Unlock()
	if status.lastSeen == timestamp.InfiniteFuture {
		m.activeEndpoints[*ep] = &containerEndpointIndicatorWithAge{
			indicator,
			lastUpdate,
		}
		return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpSuccessActive, reasonPLOP
	}
	return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpSuccessInactive, reasonPLOP
}

func (m *networkFlowManager) enrichPLOP(
	ep *containerEndpoint,
	container clusterentities.ContainerMetadata,
	processesListening map[processListeningIndicator]timestamp.MicroTS,
	lastSeen timestamp.MicroTS) (resultPLOP EnrichmentResult, reasonPLOP EnrichmentReasonEp) {
	if ep.processKey == emptyProcessInfo {
		return EnrichmentResultInvalidInput, EnrichmentReasonEpEmptyProcessInfo
	}
	indicatorPLOP := processListeningIndicator{
		key: processUniqueKey{
			podID:         container.PodID,
			containerName: container.ContainerName,
			deploymentID:  container.DeploymentID,
			process:       ep.processKey,
		},
		port:      ep.endpoint.IPAndPort.Port,
		protocol:  ep.endpoint.L4Proto.ToProtobuf(),
		podUID:    container.PodUID,
		namespace: container.Namespace,
	}
	processesListening[indicatorPLOP] = lastSeen
	return EnrichmentResultSuccess, EnrichmentReasonEp("")
}

// deactivateEndpointNoLock removes endpoint from active endpoints and sets the timestamp in enrichedEndpoints.
// It returns error when endpoint is not found in active endpoints.
func deactivateEndpointNoLock(ep *containerEndpoint,
	activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge,
	enrichedEndpoints map[containerEndpointIndicator]timestamp.MicroTS,
	now timestamp.MicroTS) bool {
	activeEp, found := activeEndpoints[*ep]
	if !found {
		return false // endpoint rotten
	}
	// Active endpoint found for historical container => removing from active endpoints and setting last-seen.
	enrichedEndpoints[activeEp.containerEndpointIndicator] = now
	delete(activeEndpoints, *ep)
	flowMetrics.SetActiveFlowsTotalGauge(len(activeEndpoints))
	return true
}

// handleConnectionEnrichmentResult prints user-readable logs explaining the result of the enrichments and returns an action
// to execute after the enrichment.
func (m *networkFlowManager) handleEndpointEnrichmentResult(
	resultNG EnrichmentResult, resultPLOP EnrichmentResult,
	reasonNG EnrichmentReasonEp, reasonPLOP EnrichmentReasonEp,
	ep *containerEndpoint) PostEnrichmentAction {

	// Currently, PLoP enrichment result alone would never cause a RetryLater action, as the part of the code
	// that may lead to retries is shared and executed before the PLoP enrichment.
	// All actions in PLoP enrichment path lead currently to PostEnrichmentActionRemove, so it is sufficient that
	// the final action is computed based on `resultNG`.
	// Here, we analyze `resultPLOP` to provide informative debug logs.
	switch resultPLOP {
	case EnrichmentResultSuccess:
		log.Debugf("PLoP enrichment succeeded")
	case EnrichmentResultSkipped:
		log.Debugf("PLoP enrichment skipped: %s", reasonPLOP)
	case EnrichmentResultInvalidInput:
		log.Debugf("Incomplete data for PLoP enrichment: %s", reasonPLOP)
	}

	switch resultNG {
	case EnrichmentResultContainerIDMissMarkRotten:
		// Endpoint cannot be expired (not found in activeConnections) and ContainerID is unknown.
		// We mark that as rotten, so that it is removed from hostConnections and not retried anymore.
		log.Debugf("ContainerID %s unknown for inactive endpoint. Marking as rotten.", ep.containerID)
		return PostEnrichmentActionRemove
	case EnrichmentResultContainerIDMissMarkInactive:
		log.Debugf("ContainerID %s unknown for active endpoint. Marking as inactive.", ep.containerID)
		return PostEnrichmentActionMarkInactive
	case EnrichmentResultRetryLater:
		switch reasonNG {
		case EnrichmentReasonEpStillInGracePeriod:
			log.Debugf("ContainerID %s unknown for active endpoint. Will retry later.", ep.containerID)
		}
		return PostEnrichmentActionRetry
	case EnrichmentResultInvalidInput:
		switch reasonNG {
		case EnrichmentReasonEpEmptyProcessInfo:
			log.Debugf("Not enriching for processes listening: empty process info")
			// PLoP enrichment cannot proceed, but we still must make sure that the enrichment for network graph is done.
			// CheckRemove will check that and decide whether to retry the enrichment or remove the endpoint from hostConns.
			return PostEnrichmentActionCheckRemove
		default:
			log.Debugf("Incomplete data to do the enrichment")
			return PostEnrichmentActionRemove
		}
	case EnrichmentResultSuccess:
		switch reasonNG {
		case EnrichmentReasonEpSuccessActive:
			log.Debugf("Enrichment succeeded; marking endpoint as active")
		case EnrichmentReasonEpSuccessInactive:
			log.Debugf("Enrichment succeeded; marking endpoint as inactive")
		case EnrichmentReasonEpDuplicate:
			log.Debugf("Enrichment succeeded; skipping update as newer data is already available")
		case EnrichmentReasonEpFeatureDisabled:
			log.Debugf("Enrichment succeeded; skipping update as sensor is not configured to enrich events while in offline mode")
		}
		// The default action is the old behavior, in which only inactive connections are removed.
		return PostEnrichmentActionCheckRemove
	default:
		log.Panicf("Programmer error: Unknown enrichment resultNG received: %v", resultNG)
		return PostEnrichmentActionCheckRemove
	}
}

func updateEndpointMetric(now timestamp.MicroTS,
	action PostEnrichmentAction,
	result EnrichmentResult, resultPLOP EnrichmentResult,
	reason EnrichmentReasonEp, reasonPLOP EnrichmentReasonEp, status *connStatus) {
	flowMetrics.FlowEnrichmentEventsEndpoint.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(status.containerIDFound),
		"result":           string(result),
		"action":           string(action),
		"isHistorical":     strconv.FormatBool(status.historicalContainerID),
		"reason":           string(reason),
		"lastSeenSet":      strconv.FormatBool(status.lastSeen < timestamp.InfiniteFuture),
		"rotten":           strconv.FormatBool(status.rotten),
		"mature":           strconv.FormatBool(status.pastContainerResolutionDeadline(now)),
		"fresh":            strconv.FormatBool(status.isFresh(now))},
	).Inc()

	flowMetrics.HostProcessesEnrichmentEvents.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(status.containerIDFound),
		"result":           string(resultPLOP),
		"action":           string(action),
		"isHistorical":     strconv.FormatBool(status.historicalContainerID),
		"reason":           string(reasonPLOP),
		"lastSeenSet":      strconv.FormatBool(status.lastSeen < timestamp.InfiniteFuture),
		"rotten":           strconv.FormatBool(status.rotten),
		"mature":           strconv.FormatBool(status.pastContainerResolutionDeadline(now)),
		"fresh":            strconv.FormatBool(status.isFresh(now))},
	).Inc()
}
