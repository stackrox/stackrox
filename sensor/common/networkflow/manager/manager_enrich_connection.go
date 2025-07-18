package manager

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	flowMetrics "github.com/stackrox/rox/sensor/common/networkflow/metrics"
)

// EnrichmentReasonConn provides additional information about given EnrichmentResult for Connections
type EnrichmentReasonConn string

const (
	EnrichmentReasonConnSuccess                    EnrichmentReasonConn = "success"
	EnrichmentReasonConnStillInGracePeriod         EnrichmentReasonConn = "still-in-grace-period"
	EnrichmentReasonConnOutsideOfGracePeriod       EnrichmentReasonConn = "outside-of-grace-period"
	EnrichmentReasonConnIPUnresolvableYet          EnrichmentReasonConn = "ip-unresolvable-yet"
	EnrichmentReasonConnLookupByNetworkFailed      EnrichmentReasonConn = "lookup-by-network-failed"
	EnrichmentReasonConnParsingIPFailed            EnrichmentReasonConn = "parsing-ip-failed"
	EnrichmentReasonConnIncomingInternalConnection EnrichmentReasonConn = "incoming-internal-connection"
)

func (m *networkFlowManager) enrichHostConnections(now timestamp.MicroTS, hostConns *hostConnections, enrichedConnections map[networkConnIndicator]timestamp.MicroTS) {
	hostConns.mutex.Lock()
	defer hostConns.mutex.Unlock()

	flowMetrics.HostConnectionsOperations.WithLabelValues("enrich", "connections").Add(float64(len(hostConns.connections)))
	for conn, status := range hostConns.connections {
		result, reason := m.enrichConnection(now, &conn, status, enrichedConnections)
		action := m.handleConnectionEnrichmentResult(result, reason, conn)
		updateConnectionMetric(now, action, result, reason, status)
		switch action {
		case PostEnrichmentActionRemove:
			delete(hostConns.connections, conn)
			flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "connections").Inc()
		case PostEnrichmentActionMarkInactive:
			concurrency.WithLock(&m.activeConnectionsMutex, func() {
				if ok := deactivateConnectionNoLock(&conn, m.activeConnections, enrichedConnections, now); !ok {
					log.Debugf("Cannot mark connection as inactive: connection is rotten")
				}
			})
		case PostEnrichmentActionRetry:
		// noop, retry happens through not removing from `hostConns.connections`
		case PostEnrichmentActionCheckRemove:
			if status.isClosed() {
				delete(hostConns.connections, conn)
				flowMetrics.HostConnectionsOperations.WithLabelValues("remove", "connections").Inc()
			}
		default:
			log.Warnf("Unknown enrichment action: %v", action)
		}
	}
}

// enrichConnection updates `enrichedConnections` and `m.activeConnections`.
// It "returns" the outcome of the enrichment in the `status`.
func (m *networkFlowManager) enrichConnection(now timestamp.MicroTS, conn *connection, status *connStatus, enrichedConnections map[networkConnIndicator]timestamp.MicroTS) (EnrichmentResult, EnrichmentReasonConn) {
	timeElapsedSinceFirstSeen := now.ElapsedSince(status.firstSeen)
	pastContainerResolutionDeadline := timeElapsedSinceFirstSeen > env.ContainerIDResolutionGracePeriod.DurationSetting()
	isFresh := timeElapsedSinceFirstSeen < clusterEntityResolutionWaitPeriod

	container, contIDfound, isHistorical := m.clusterEntities.LookupByContainerID(conn.containerID)
	status.historicalContainerID = isHistorical
	status.containerIDFound = contIDfound
	if !contIDfound {
		// There is a connection involving a container that Sensor does not recognize. In this case we may do two things:
		// (1) decide that we want to retry the enrichment later (keep the connection in hostConnections),
		// (2) remove the connection from hostConnections, because enrichment is impossible.
		if !pastContainerResolutionDeadline {
			return EnrichmentResultRetryLater, EnrichmentReasonConnStillInGracePeriod
		}
		// Expire the connection if we are past the containerID resolution grace period.
		result := concurrency.WithLock1(&m.activeConnectionsMutex, func() EnrichmentResult {
			if _, found := m.activeConnections[*conn]; !found {
				return EnrichmentResultContainerIDMissMarkRotten
			}
			return EnrichmentResultContainerIDMissMarkInactive
		})

		return result, EnrichmentReasonConnOutsideOfGracePeriod
	}

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

	if len(lookupResults) == 0 {
		// If the address is set and is not resolvable, we want to we wait for `clusterEntityResolutionWaitPeriod` time
		// before associating it to a known network or INTERNET.
		if isFresh && conn.remote.IPAndPort.Address.IsValid() {
			return EnrichmentResultRetryLater, EnrichmentReasonConnIPUnresolvableYet
		}

		extSrc := m.externalSrcs.LookupByNetwork(conn.remote.IPAndPort.IPNetwork)
		if extSrc != nil {
			// Network was resolved, so there is no need to retry that
			isFresh = false
		}

		if isFresh { // FIXME: Refactor this condition
			return EnrichmentResultRetryLater, EnrichmentReasonConnLookupByNetworkFailed
		}

		defer func() {
			status.enrichmentResult.consumedNetworkGraph = true
		}()

		if extSrc == nil {
			entityType := networkgraph.InternetEntity()
			isExternal, err := conn.IsExternal()
			status.isExternal = isExternal
			if err != nil {
				// IP is malformed or unknown - do not show on the graph and log the info
				// TODO(ROX-22388): Change log level back to warning when potential Collector issue is fixed
				log.Debugf("Enrichment aborted: %v", err)
				return EnrichmentResultInvalidInput, EnrichmentReasonConnParsingIPFailed
			}
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

			if !status.enrichmentResult.consumedNetworkGraph {
				// Count internal metrics even if central lacks `NetworkGraphInternalEntitiesSupported` capability.
				if isExternal {
					flowMetrics.ExternalFlowCounter.With(metricDirection).Inc()
				} else {
					flowMetrics.InternalFlowCounter.With(metricDirection).Inc()
				}
			}
		} else {
			if !status.enrichmentResult.consumedNetworkGraph {
				flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
			}
			lookupResults = []clusterentities.LookupResult{
				{
					Entity:         networkgraph.EntityFromProto(extSrc),
					ContainerPorts: []uint16{port},
				},
			}
		}
	} else {
		if !status.enrichmentResult.consumedNetworkGraph {
			flowMetrics.NetworkEntityFlowCounter.With(metricDirection).Inc()
		}
		status.enrichmentResult.consumedNetworkGraph = true
		if conn.incoming {
			// Endpoint lookup successful, connection is incoming and local.
			// Skip enrichment for this connection, as it is already taken care of by
			// the corresponding outgoing connection in opposite direction.
			return EnrichmentResultInvalidInput, EnrichmentReasonConnIncomingInternalConnection
		}
	}

	for _, lookupResult := range lookupResults {
		for _, port := range lookupResult.ContainerPorts {
			indicator := networkConnIndicatorWithAge{
				networkConnIndicator: networkConnIndicator{
					dstPort:  port,
					protocol: conn.remote.L4Proto.ToProtobuf(),
				},
				lastUpdate: now,
			}

			if conn.incoming {
				indicator.srcEntity = lookupResult.Entity
				indicator.dstEntity = networkgraph.EntityForDeployment(container.DeploymentID)
			} else {
				indicator.srcEntity = networkgraph.EntityForDeployment(container.DeploymentID)
				indicator.dstEntity = lookupResult.Entity
			}

			// Multiple connections from a collector can result in a single enriched connection
			// hence update the timestamp only if we have a more recent connection than the one we have already enriched.
			if oldTS, found := enrichedConnections[indicator.networkConnIndicator]; !found || oldTS < status.lastSeen {
				enrichedConnections[indicator.networkConnIndicator] = status.lastSeen
				if !features.SensorCapturesIntermediateEvents.Enabled() {
					continue
				}

				concurrency.WithLock(&m.activeConnectionsMutex, func() {
					if status.lastSeen == timestamp.InfiniteFuture {
						m.activeConnections[*conn] = &indicator
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

// handleConnectionEnrichmentResult prints user-readable logs explaining the result of the enrichments and returns an action
// to execute after the enrichment.
func (m *networkFlowManager) handleConnectionEnrichmentResult(result EnrichmentResult, reason EnrichmentReasonConn, conn connection) PostEnrichmentAction {
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
		case EnrichmentReasonConnIncomingInternalConnection:
			// Endpoint lookup successful, connection is incoming and local.
			// Skip enrichment for this connection, as it is already taken care of by
			// the corresponding outgoing connection in opposite direction.
			// No need to log.
		}
		return PostEnrichmentActionRetry
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
	enrichedConnections map[networkConnIndicator]timestamp.MicroTS,
	now timestamp.MicroTS,
) bool {
	activeConn, found := activeConnections[*conn]
	if !found {
		// Connection is rotten
		return false
	}
	// Active connection found - mark that Sensor considers this connection no longer active
	// due to missing data about the container.
	enrichedConnections[activeConn.networkConnIndicator] = now
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
		"lastSeenSet":      strconv.FormatBool(status.lastSeen < timestamp.InfiniteFuture),
		"rotten":           strconv.FormatBool(status.rotten),
		"mature":           strconv.FormatBool(status.pastContainerResolutionDeadline(now)),
		"fresh":            strconv.FormatBool(status.isFresh(now)),
		"isExternal":       strconv.FormatBool(status.isExternal),
	}).Inc()
}
