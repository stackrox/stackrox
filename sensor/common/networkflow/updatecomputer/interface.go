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
	ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint
	ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor

	// OnSuccessfulSend contains actions that should be executed after successful sending of updates to Central.
	OnSuccessfulSend(currentConns map[indicator.NetworkConn]timestamp.MicroTS,
		currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS,
		currentProcesses map[indicator.ProcessListening]timestamp.MicroTS)

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
