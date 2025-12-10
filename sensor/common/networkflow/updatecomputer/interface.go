package updatecomputer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// UpdateComputer defines the interface for computing network flow updates sent to Central.
type UpdateComputer interface {
	// ComputeUpdatedConns updates based on currentState state and implementation-specific tracking
	ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow
	ComputeUpdatedEndpointsAndProcesses(
		enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp,
	) ([]*storage.NetworkEndpoint, []*storage.ProcessListeningOnPortFromSensor)

	// OnSuccessfulSendConnections contains actions that should be executed after successful sending of connection updates to Central.
	OnSuccessfulSendConnections(currentConns map[indicator.NetworkConn]timestamp.MicroTS)
	// OnSuccessfulSendEndpoints contains actions that should be executed after successful sending of endpoint updates to Central.
	OnSuccessfulSendEndpoints(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp)
	// OnSuccessfulSendProcesses contains actions that should be executed after successful sending of processesListening updates to Central.
	OnSuccessfulSendProcesses(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp)
	// OnSendConnectionsFailure is called after sending with connections that failed to send. These items are stored in cache for retry.
	OnSendConnectionsFailure(unsentConns []*storage.NetworkFlow)
	// OnSendEndpointsFailure is called after sending with endpoints that failed to send. These items are stored in cache for retry.
	OnSendEndpointsFailure(unsentEps []*storage.NetworkEndpoint)
	// OnSendProcessesFailure is called after sending with processes that failed to send. These items are stored in cache for retry.
	OnSendProcessesFailure(unsentProcs []*storage.ProcessListeningOnPortFromSensor)

	// ResetState resets all internal state (used when clearing historical data).
	ResetState()

	// PeriodicCleanup should be run periodically to clean up the temporal data.
	PeriodicCleanup(now time.Time, cleanupInterval time.Duration)

	// RecordSizeMetrics records metrics for length and byte-size of the collections used in updateComputer.
	RecordSizeMetrics(gv1, gv2 *prometheus.GaugeVec)
}

func New() UpdateComputer {
	if env.NetworkFlowUseLegacyUpdateComputer.BooleanSetting() {
		return NewLegacy()
	}
	return NewTransitionBased()
}
