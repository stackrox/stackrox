package updatecomputer

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// UpdateComputer defines the interface for computing network flow updates to send to Central
// Each implementation manages its own state and computation strategy
type UpdateComputer interface {
	// ComputeUpdatedConns updates based on currentState state and implementation-specific tracking
	ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) ([]*storage.NetworkFlow, error)
	ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint
	ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor

	// UpdateState covers state management - each implementation handles its own state updates
	UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[indicator.ProcessListening]timestamp.MicroTS)

	// ResetState resets all internal state (used when clearing historical data)
	ResetState()

	// PeriodicCleanup should be run periodically to clean up the temporal data.
	PeriodicCleanup(now time.Time, cleanupInterval time.Duration)

	// GetStateMetrics returns metric values about internal state size for monitoring
	GetStateMetrics() map[string]map[string]int
}
