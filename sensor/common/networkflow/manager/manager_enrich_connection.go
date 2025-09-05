package manager

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

// executeConnectionAction performs the specified post-enrichment action on a connection
// and returns the removeCheckResult for metrics tracking.
func (m *networkFlowManager) executeConnectionAction(
	action PostEnrichmentAction,
	conn *connection,
	status *connStatus,
	hostConns *hostConnections,
	enrichedConnections map[indicator.NetworkConn]timestamp.MicroTS,
	now timestamp.MicroTS,
) {
	switch action {
	case PostEnrichmentActionRemove:
		delete(hostConns.connections, *conn)
		flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "connections").Inc()
	case PostEnrichmentActionMarkInactive:
		concurrency.WithLock(&m.activeConnectionsMutex, func() {
			if ok := deactivateConnectionNoLock(conn, m.activeConnections, enrichedConnections, now); !ok {
				log.Debugf("Cannot mark connection as inactive: connection is rotten")
			}
		})
	case PostEnrichmentActionRetry:
		// noop, retry happens through not removing from `hostConns.connections`
	case PostEnrichmentActionCheckRemove:
		if status.rotten || status.enrichmentConsumption.consumedNetworkGraph {
			delete(hostConns.connections, *conn)
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "connections").Inc()
		}
	default:
		log.Warnf("Unknown enrichment action: %v", action)
	}
}

func (m *networkFlowManager) enrichHostConnections(now timestamp.MicroTS, hostConns *hostConnections, enrichedConnections map[indicator.NetworkConn]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	flowMetrics.HostConnectionsOperations.WithLabelValues("enrich", "connections").Add(float64(len(hostConns.connections)))
	for conn, status := range hostConns.connections {
		result, reason := m.enrichConnection(now, &conn, status, enrichedConnections)
		action := m.handleConnectionEnrichmentResult(result, reason, &conn)
		m.executeConnectionAction(action, &conn, status, hostConns, enrichedConnections, now)
		updateConnectionMetric(now, action, result, reason, status)
	}
}

// enrichConnection updates `enrichedConnections` and `m.activeConnections`.
// It returns the enrichment result and provides reason for returning such result.
// Additionally, it sets the outcome in the `status` field to reflect the outcome of the enrichment in memory-efficient way by avoiding copying.
func (m *networkFlowManager) enrichConnection(now timestamp.MicroTS, conn *connection, status *connStatus, enrichedConnections map[indicator.NetworkConn]timestamp.MicroTS) (EnrichmentResult, EnrichmentReasonConn) {
	isFresh := status.isFresh(now)

	// Use shared container resolution logic
	activeChecker := &connectionActiveChecker{mutex: &m.activeConnectionsMutex, activeConnections: m.activeConnections}
	containerResult := resolveContainerID(m, now, conn.containerID, status, activeChecker, *conn)

	if !containerResult.Found {
		// There is a connection involving a container that Sensor does not recognize. In this case we may do two things:
		// (1) decide that we want to retry the enrichment later (keep the connection in hostConnections)
		// - this is done while still within the containerID resolution grace period,
		// (2) remove the connection from hostConnections, because enrichment is impossible
		// - this is done after the containerID resolution grace period.
		if containerResult.ShouldRetryLater {
			return EnrichmentResultRetryLater, EnrichmentReasonConnStillInGracePeriod
		}
		// Expire the connection if we are past the containerID resolution grace period.
		return containerResult.DeactivationResult, EnrichmentReasonConnOutsideOfGracePeriod
	}

	container := containerResult.Container

	var lookupResults []clusterentities.LookupResult
	var isInternet = false

	// Check if the remote address represents the de-facto INTERNET entity.
	if conn.remote.IsConsideredExternal() {
		isFresh = false
		isInternet = true
	} else {
		// Otherwise, check if the remote entity is actually a cluster entity.
		lookupResults = m.clusterEntities.LookupByEndpoint(conn.remote)
	}

	var port uint16
	var direction string
	if conn.incoming {
		direction = "ingress"
		port = conn.local.Port
	} else {
		direction = "egress"
		port = conn.remote.IPAndPort.Port
	}

	metricDirection := prometheus.Labels{
		"direction": direction,
		"namespace": container.Namespace,
	}

	// Cannot find any entity when looking by endpoint and IP address.
	if len(lookupResults) == 0 {
		// If the address is set and is not resolvable, we want to we wait for `clusterEntityResolutionWaitPeriod` time
		// before associating it to a known network or INTERNET.
		if isFresh && conn.remote.IPAndPort.Address.IsValid() {
			return EnrichmentResultRetryLater, EnrichmentReasonConnIPUnresolvableYet
		}

		externalSource := m.externalSrcs.LookupByNetwork(conn.remote.IPAndPort.IPNetwork)
		// If we're still within the cluster entity resolution wait period and couldn't find
		// a matching external network source, then we retry later. This gives time for external
		// network definitions to be loaded before falling back to generic entities (Internet or Internal).
		if isFresh && externalSource == nil {
			return EnrichmentResultRetryLater, EnrichmentReasonConnLookupByNetworkFailed
		}

		defer func() {
			status.enrichmentConsumption.consumedNetworkGraph = true
		}()

		if externalSource == nil {
			entityType := networkgraph.InternetEntity()
			isExternal, err := conn.IsExternal()
			if err != nil {
				// IP is malformed or unknown - do not show on the graph and log the info
				// TODO(ROX-22388): Change log level back to warning when potential Collector issue is fixed
				log.Debugf("Enrichment aborted: %v", err)
				return EnrichmentResultInvalidInput, EnrichmentReasonConnParsingIPFailed
			}
			status.isExternal = isExternal
			if isExternal {
				// If Central does not handle DiscoveredExternalEntities, report an Internet entity as it used to be.
				if !isInternet && centralcaps.Has(centralsensor.NetworkGraphDiscoveredExternalEntitiesSupported) {
					entityType = networkgraph.DiscoveredExternalEntity(net.IPNetworkFromNetworkPeerID(conn.remote.IPAndPort))
				}
			} else if centralcaps.Has(centralsensor.NetworkGraphInternalEntitiesSupported) {
				// Central without the capability would crash the UI if we make it display "Internal Entities".
				entityType = networkgraph.InternalEntities()
			}

			// Fake a lookup result. This shows "External Entities" or "Internal Entities" in the network graph
			lookupResults = []clusterentities.LookupResult{
				{
					Entity:         entityType,
					ContainerPorts: []uint16{port},
				},
			}
			entitiesName := "Internal Entities"
			if isExternal {
				entitiesName = "External Entities"
			}
			logReasonForAggregatingNetGraphFlow(conn, container.Namespace, container.ContainerName, entitiesName, port)

			if !status.enrichmentConsumption.consumedNetworkGraph {
				// Count internal metrics even if central lacks `NetworkGraphInternalEntitiesSupported` capability.
				if isExternal {
					flowMetrics.ExternalFlowCounter.With(metricDirection).Inc()
				} else {
					flowMetrics.InternalFlowCounter.With(metricDirection).Inc()
				}
			}
		} else {
			if !status.enrichmentConsumption.consumedNetworkGraph {
				flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
			}
			lookupResults = []clusterentities.LookupResult{
				{
					Entity:         networkgraph.EntityFromProto(externalSource),
					ContainerPorts: []uint16{port},
				},
			}
		}
	} else {
		if !status.enrichmentConsumption.consumedNetworkGraph {
			flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
		}
		status.enrichmentConsumption.consumedNetworkGraph = true
		if conn.incoming {
			// Endpoint lookup successful, connection is incoming and local.
			// Skip enrichment for this connection, as it is already taken care of by
			// the corresponding outgoing connection in opposite direction.
			return EnrichmentResultSkipped, EnrichmentReasonConnIncomingInternalConnection
		}
	}

	for _, lookupResult := range lookupResults {
		for _, port := range lookupResult.ContainerPorts {
			ind := networkConnIndicatorWithAge{
				NetworkConn: indicator.NetworkConn{
					DstPort:  port,
					Protocol: conn.remote.L4Proto.ToProtobuf(),
				},
				lastUpdate: now,
			}

			if conn.incoming {
				ind.SrcEntity = lookupResult.Entity
				ind.DstEntity = networkgraph.EntityForDeployment(container.DeploymentID)
			} else {
				ind.SrcEntity = networkgraph.EntityForDeployment(container.DeploymentID)
				ind.DstEntity = lookupResult.Entity
			}

			// Multiple connections from a collector can result in a single enriched connection
			// hence update the timestamp only if we have a more recent connection than the one we have already enriched.
			if oldTS, found := enrichedConnections[ind.NetworkConn]; !found || oldTS < status.lastSeen {
				enrichedConnections[ind.NetworkConn] = status.lastSeen
				if !features.SensorCapturesIntermediateEvents.Enabled() {
					continue
				}

				concurrency.WithLock(&m.activeConnectionsMutex, func() {
					if !status.isClosed() {
						m.activeConnections[*conn] = &ind
						flowMetrics.SetActiveFlowsTotalGauge(len(m.activeConnections))
						return
					}
					delete(m.activeConnections, *conn)
					flowMetrics.SetActiveFlowsTotalGauge(len(m.activeConnections))
				})
			}
		}
	}
	return EnrichmentResultSuccess, EnrichmentReasonConnSuccess
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

// handleConnectionEnrichmentResult prints user-readable logs explaining the result of the enrichments and returns an action
// to execute after the enrichment.
func (m *networkFlowManager) handleConnectionEnrichmentResult(result EnrichmentResult, reason EnrichmentReasonConn, conn *connection) PostEnrichmentAction {
	switch result {
	case EnrichmentResultContainerIDMissMarkRotten:
		// Connection cannot be expired (not contIDfound in activeConnections) and ContainerID is unknown.
		// We mark that as rotten, so that it is removed from hostConnections and not retried anymore.
		log.Debugf("ContainerID %s unknown for inactive connection. Marking as rotten", conn.containerID)
		return PostEnrichmentActionRemove
	case EnrichmentResultContainerIDMissMarkInactive:
		log.Debugf("ContainerID %s unknown for active connection. Marking as inactive.", conn.containerID)
		return PostEnrichmentActionMarkInactive
	case EnrichmentResultRetryLater:
		switch reason {
		case EnrichmentReasonConnStillInGracePeriod:
			log.Debugf("ContainerID %s unknown for active connection. Will retry later.", conn.containerID)
		case EnrichmentReasonConnIPUnresolvableYet:
			log.Debugf("Unable to resolve address %q. Will retry later.", conn.remote.String())
		case EnrichmentReasonConnLookupByNetworkFailed:
			log.Debugf("Unknown external network %q. Will retry later.", conn.remote.IPAndPort.IPNetwork.String())
		}
		return PostEnrichmentActionRetry
	case EnrichmentResultInvalidInput:
		switch reason {
		case EnrichmentReasonConnParsingIPFailed:
			log.Debugf("Enrichment failed. Unable to parse IP address.")
			return PostEnrichmentActionRetry
		}
	case EnrichmentResultSkipped:
		switch reason {
		case EnrichmentReasonConnIncomingInternalConnection:
			// Endpoint lookup successful, connection is incoming and local.
			// Skip enrichment for this connection, as it is already taken care of by
			// the corresponding outgoing connection in opposite direction.
			// No need to log.
			log.Debugf("Enrichment skipped. Connection is incoming and local.")
			return PostEnrichmentActionRemove
		}
		return PostEnrichmentActionCheckRemove
	case EnrichmentResultSuccess:
		// no log, default action for successful enrichment
		return PostEnrichmentActionCheckRemove
	}
	log.Panicf("Programmer error: Unknown enrichment result received: %v", result)
	// Safest choice for default action
	return PostEnrichmentActionCheckRemove
}

// deactivateConnectionNoLock is executed when Sensor decides that the full enrichment cannot be successful
// and further enrichments shouldn't be attempted unless new data (containerID) arrives from k8s.
// Returns true when connection was removed from activeConnections, and false if not found within activeConnections.
func deactivateConnectionNoLock(conn *connection,
	activeConnections map[connection]*networkConnIndicatorWithAge,
	enrichedConnections map[indicator.NetworkConn]timestamp.MicroTS,
	now timestamp.MicroTS,
) bool {
	activeConn, found := activeConnections[*conn]
	if !found {
		// Connection is rotten
		return false
	}
	// Active connection found - mark that Sensor considers this connection no longer active
	// due to missing data about the container.
	enrichedConnections[activeConn.NetworkConn] = now
	delete(activeConnections, *conn)
	flowMetrics.SetActiveFlowsTotalGauge(len(activeConnections))
	return true
}

func updateConnectionMetric(now timestamp.MicroTS, action PostEnrichmentAction, result EnrichmentResult, reason EnrichmentReasonConn, status *connStatus) {
	flowMetrics.FlowEnrichmentEventsConnection.With(prometheus.Labels{
		"containerIDfound": strconv.FormatBool(status.containerIDFound),
		"result":           string(result),
		"action":           string(action),
		"isHistorical":     strconv.FormatBool(status.historicalContainerID),
		"reason":           string(reason),
		"isClosed":         strconv.FormatBool(status.isClosed()),
		"rotten":           strconv.FormatBool(status.rotten),
		"mature":           strconv.FormatBool(status.pastContainerResolutionDeadline(now)),
		"fresh":            strconv.FormatBool(status.isFresh(now)),
		"isExternal":       strconv.FormatBool(status.isExternal),
	}).Inc()
}
