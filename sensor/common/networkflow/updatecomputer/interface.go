package updatecomputer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// UpdateComputerType represents the type of update computer to use
type UpdateComputerType string

const (
	// LegacyUpdateComputerType uses the original Legacy LastSentState-based logic
	LegacyUpdateComputerType UpdateComputerType = "legacy"
	// CategorizedUpdateComputerType uses the new Categorized update logic
	CategorizedUpdateComputerType UpdateComputerType = "categorized"
)

// UpdateComputer defines the interface for computing network flow updates to send to Central
// Each implementation manages its own state and computation strategy
type UpdateComputer interface {
	// Compute updates based on current state and implementation-specific tracking
	ComputeUpdatedConns(current map[*indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow
	ComputeUpdatedEndpoints(current map[*indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint
	ComputeUpdatedProcesses(current map[*indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor

	// State management - each implementation handles its own state updates
	UpdateState(currentConns map[*indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[*indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[*indicator.ProcessListening]timestamp.MicroTS)

	// Reset all internal state (used when clearing historical data)
	ResetState()

	// Get metrics about internal state size for monitoring
	GetStateMetrics() (connsSize, endpointsSize, processesSize int)
}
