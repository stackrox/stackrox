package manager

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

// executeEndpointAction performs the specified post-enrichment action on an endpoint
// and returns the removeCheckResult for metrics tracking.
func (m *networkFlowManager) executeEndpointAction(
	action PostEnrichmentAction,
	ep *containerEndpoint,
	status *connStatus,
	hostConns *hostConnections,
	enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp,
	now timestamp.MicroTS,
) {
	switch action {
	case PostEnrichmentActionRemove:
		delete(hostConns.endpoints, *ep)
		flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "endpoints").Inc()
	case PostEnrichmentActionMarkInactive:
		concurrency.WithLock(&m.activeEndpointsMutex, func() {
			if ok := deactivateEndpointNoLock(ep, m.activeEndpoints, enrichedEndpointsProcesses, now); !ok {
				log.Debugf("Cannot mark endpoint as inactive: endpoint is rotten")
			}
		})
	case PostEnrichmentActionRetry:
		// noop, retry happens through not removing from `hostConns.endpoints`
	case PostEnrichmentActionCheckRemove:
		if status.checkRemoveCondition(env.NetworkFlowUseLegacyUpdateComputer.BooleanSetting(), status.enrichmentConsumption.IsConsumed()) {
			delete(hostConns.endpoints, *ep)
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "endpoints").Inc()
			flowMetrics.HostProcessesEvents.WithLabelValues("remove").Inc()
		}
	default:
		log.Warnf("Unknown enrichment action: %v", action)
	}
}

func (m *networkFlowManager) enrichHostContainerEndpoints(now timestamp.MicroTS, hostConns *hostConnections,
	enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
	concurrency.WithLock(&hostConns.mutex, func() {
		flowMetrics.HostProcessesEvents.WithLabelValues("add").Add(float64(len(hostConns.endpoints)))
		flowMetrics.HostConnectionsOperations.WithLabelValues("enrich", "endpoints").Add(float64(len(hostConns.endpoints)))
		for ep, status := range hostConns.endpoints {
			resultNG, resultPLOP, reasonNG, reasonPLOP := m.enrichContainerEndpoint(now, &ep, status, enrichedEndpointsProcesses, now)
			action := m.handleEndpointEnrichmentResult(resultNG, resultPLOP, reasonNG, reasonPLOP, &ep)
			m.executeEndpointAction(action, &ep, status, hostConns, enrichedEndpointsProcesses, now)
			updateEndpointMetric(now, action, resultNG, resultPLOP, reasonNG, reasonPLOP, status)
		}
	})
	concurrency.WithRLock(&m.activeEndpointsMutex, func() {
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
	enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp,
	lastUpdate timestamp.MicroTS,
) (resultNG, resultPLOP EnrichmentResult, reasonNG, reasonPLOP EnrichmentReasonEp) {
	isFresh := status.isFresh(now)
	if !isFresh {
		status.enrichmentConsumption.consumedNetworkGraph = true
	}

	// Use shared container resolution logic
	activeChecker := &endpointActiveChecker{mutex: &m.activeEndpointsMutex, activeEndpoints: m.activeEndpoints}
	containerResult := resolveContainerID(m, now, ep.containerID, status, activeChecker, *ep)

	if !containerResult.Found {
		// There is an endpoint involving a container that Sensor does not recognize. In this case we may do two things:
		// (1) decide that we want to retry the enrichment later (keep the endpoint in hostConnections)
		// - this is done while still within the containerID resolution grace period,
		// (2) remove the endpoint from hostConnections, because enrichment is impossible
		// - this is done after the containerID resolution grace period.
		if containerResult.ShouldRetryLater {
			return EnrichmentResultRetryLater, EnrichmentResultRetryLater,
				EnrichmentReasonEpStillInGracePeriod, EnrichmentReasonEpStillInGracePeriod
		}
		// Expire the connection if we are past the containerID resolution grace period.
		return containerResult.DeactivationResult, containerResult.DeactivationResult,
			EnrichmentReasonEpOutsideOfGracePeriod, EnrichmentReasonEpOutsideOfGracePeriod
	}

	container := containerResult.Container
	processIndicator := &indicator.ProcessListeningWithTimestamp{
		ProcessListening: nil,
		LastSeen:         status.lastSeen,
	}

	// SECTION: ENRICHMENT OF PROCESSES LISTENING ON PORTS
	if env.ProcessesListeningOnPort.BooleanSetting() {
		status.enrichmentConsumption.consumedPLOP = true
		var pi *indicator.ProcessListening
		pi, resultPLOP, reasonPLOP = m.enrichPLOP(ep, container)
		// Always store processIndicator, even if nil.
		processIndicator.ProcessListening = pi
	} else {
		resultPLOP = EnrichmentResultSkipped
		reasonPLOP = EnrichmentReasonEpFeaturePlopDisabled
	}

	// SECTION: ENRICHMENT OF ENDPOINT
	status.enrichmentConsumption.consumedNetworkGraph = true
	ind := indicator.ContainerEndpoint{
		Entity:   networkgraph.EntityForDeployment(container.DeploymentID),
		Port:     ep.endpoint.IPAndPort.Port,
		Protocol: ep.endpoint.L4Proto.ToProtobuf(),
	}

	// Multiple endpoints from a collector can result in a single enriched endpoint,
	// hence update the timestamp only if we have a more recent endpoint than the one we have already enriched.
	if oldValue, found := enrichedEndpointsProcesses[ind]; found && oldValue.LastSeen >= status.lastSeen {
		return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpDuplicate, reasonPLOP
	}

	enrichedEndpointsProcesses[ind] = processIndicator

	m.activeEndpointsMutex.Lock()
	defer m.activeEndpointsMutex.Unlock()
	if !status.isClosed() {
		m.activeEndpoints[*ep] = &containerEndpointIndicatorWithAge{
			ContainerEndpoint: ind,
			lastUpdate:        lastUpdate,
		}
		return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpSuccessActive, reasonPLOP
	}
	return EnrichmentResultSuccess, resultPLOP, EnrichmentReasonEpSuccessInactive, reasonPLOP
}

func (m *networkFlowManager) enrichPLOP(
	ep *containerEndpoint,
	container clusterentities.ContainerMetadata,
) (ind *indicator.ProcessListening, resultPLOP EnrichmentResult, reasonPLOP EnrichmentReasonEp) {
	if ep.processKey == emptyProcessInfo {
		return nil, EnrichmentResultInvalidInput, EnrichmentReasonEpEmptyProcessInfo
	}
	return &indicator.ProcessListening{
		PodID:         container.PodID,
		ContainerName: container.ContainerName,
		DeploymentID:  container.DeploymentID,
		Process:       ep.processKey,
		Port:          ep.endpoint.IPAndPort.Port,
		Protocol:      ep.endpoint.L4Proto.ToProtobuf(),
		PodUID:        container.PodUID,
		Namespace:     container.Namespace,
	}, EnrichmentResultSuccess, EnrichmentReasonEp("")
}

// deactivateEndpointNoLock removes endpoint from active endpoints and sets the timestamp in enrichedEndpoints.
// It returns error when endpoint is not found in active endpoints.
func deactivateEndpointNoLock(ep *containerEndpoint,
	activeEndpoints map[containerEndpoint]*containerEndpointIndicatorWithAge,
	enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp,
	now timestamp.MicroTS) bool {
	activeEp, found := activeEndpoints[*ep]
	if !found {
		return false // endpoint rotten
	}
	// Active endpoint found for historical container =>
	// (1) setting last-seen - even if not present in enrichedEndpointsProcesses and
	// (2) removing from active endpoints.
	setLastSeenOrAdd(enrichedEndpointsProcesses, activeEp.ContainerEndpoint, now)
	delete(activeEndpoints, *ep)
	flowMetrics.SetActiveEndpointsTotalGauge(len(activeEndpoints))
	return true
}

// setLastSeenOrAdd checks the map `m` for presence of `key`.
// If `key` is found, it changes the `LastSeen` to `ts` (close active endpoint, keep process as it was).
// If `key` is not found, it adds it with empty `ProcessListening` and `LastSeen` set to `ts` (artificially close active endpoint).
func setLastSeenOrAdd(m map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp, key indicator.ContainerEndpoint, ts timestamp.MicroTS) {
	if value := m[key]; value == nil {
		m[key] = &indicator.ProcessListeningWithTimestamp{
			ProcessListening: nil,
			LastSeen:         ts,
		}
	} else {
		m[key].LastSeen = ts
	}
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
		// This value is only expected by resultPLOP.
		// If (under circumstances unknown today) the resultNG contains it, we should remove the entry to prevent it
		// from piling up in the memory.
		log.Debugf("Incomplete data to do the enrichment")
		return PostEnrichmentActionRemove
	case EnrichmentResultSuccess:
		switch reasonNG {
		case EnrichmentReasonEpSuccessActive:
			log.Debugf("Enrichment succeeded; marking endpoint as active")
		case EnrichmentReasonEpSuccessInactive:
			log.Debugf("Enrichment succeeded; marking endpoint as inactive")
		case EnrichmentReasonEpDuplicate:
			log.Debugf("Enrichment succeeded; skipping update as newer data is already available")
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
	reason EnrichmentReasonEp, reasonPLOP EnrichmentReasonEp,
	status *connStatus) {
	flowMetrics.FlowEnrichmentEventsEndpoint.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(status.containerIDFound),
		"result":           string(result),
		"action":           string(action),
		"isHistorical":     strconv.FormatBool(status.historicalContainerID),
		"reason":           string(reason),
		"isClosed":         strconv.FormatBool(status.isClosed()),
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
		"isClosed":         strconv.FormatBool(status.isClosed()),
		"rotten":           strconv.FormatBool(status.rotten),
		"mature":           strconv.FormatBool(status.pastContainerResolutionDeadline(now)),
		"fresh":            strconv.FormatBool(status.isFresh(now))},
	).Inc()
}
