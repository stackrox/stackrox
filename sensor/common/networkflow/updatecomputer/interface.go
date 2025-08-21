package updatecomputer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// UpdateComputer defines the interface for computing network flow updates sent to Central.
type UpdateComputer interface {
	// ComputeUpdatedConns updates based on currentState state and implementation-specific tracking
	ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow
	ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint
	ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor

	// UpdateState brings the update computer to the desired state.
	// Required for tests and implementations that store the last sent state (e.g., Legacy).
	UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS,
		currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS,
		currentProcesses map[indicator.ProcessListening]timestamp.MicroTS)

	// ResetState resets all internal state (used when clearing historical data).
	ResetState()

	// RecordSizeMetrics records metrics for length and byte-size of the collections used in updateComputer.
	RecordSizeMetrics(name string, gv1, gv2 *prometheus.GaugeVec)
}
