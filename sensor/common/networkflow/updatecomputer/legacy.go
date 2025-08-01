package updatecomputer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

var (
	loggingRateLimiter = "plop-feature-disabled"
)

// LegacyUpdateComputer implements the original update computation logic using LastSentState maps
// It owns and manages the LastSentState maps that were previously in the manager
type LegacyUpdateComputer struct {
	// State tracking maps - these were previously in networkFlowManager
	enrichedConnsLastSentState     map[*indicator.NetworkConn]timestamp.MicroTS
	enrichedEndpointsLastSentState map[*indicator.ContainerEndpoint]timestamp.MicroTS
	enrichedProcessesLastSentState map[*indicator.ProcessListening]timestamp.MicroTS

	// Mutex to protect the LastSentState maps
	lastSentStateMutex sync.RWMutex
}

// NewLegacyUpdateComputer creates a new instance of the legacy update computer
func NewLegacyUpdateComputer() UpdateComputer {
	return &LegacyUpdateComputer{
		enrichedConnsLastSentState:     make(map[*indicator.NetworkConn]timestamp.MicroTS),
		enrichedEndpointsLastSentState: make(map[*indicator.ContainerEndpoint]timestamp.MicroTS),
		enrichedProcessesLastSentState: make(map[*indicator.ProcessListening]timestamp.MicroTS),
	}
}

func (l *LegacyUpdateComputer) ComputeUpdatedConns(current map[*indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()
	var updates []*storage.NetworkFlow

	for conn, currTS := range current {
		prevTS, seenPreviously := l.enrichedConnsLastSentState[conn]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, conn.ToProto(currTS))
		}
	}

	for conn, prevTS := range l.enrichedConnsLastSentState {
		if _, ok := current[conn]; !ok {
			updates = append(updates, conn.ToProto(prevTS))
		}
	}

	return updates
}

func (l *LegacyUpdateComputer) ComputeUpdatedEndpoints(current map[*indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()
	var updates []*storage.NetworkEndpoint

	for ep, currTS := range current {
		prevTS, seenPreviously := l.enrichedEndpointsLastSentState[ep]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, ep.ToProto(currTS))
		}
	}

	for ep, prevTS := range l.enrichedEndpointsLastSentState {
		if _, ok := current[ep]; !ok {
			updates = append(updates, ep.ToProto(prevTS))
		}
	}

	return updates
}

func (l *LegacyUpdateComputer) ComputeUpdatedProcesses(current map[*indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().Warnf(loggingRateLimiter,
				"Received %d process(es) while ProcessesListeningOnPort feature is disabled. This may indicate a misconfiguration.", len(current))
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()
	var updates []*storage.ProcessListeningOnPortFromSensor

	for pl, currTS := range current {
		prevTS, ok := l.enrichedProcessesLastSentState[pl]
		if !ok || currTS > prevTS || (prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture) {
			updates = append(updates, pl.ToProto(currTS))
		}
	}

	for ep, prevTS := range l.enrichedProcessesLastSentState {
		if _, ok := current[ep]; !ok {
			// This condition means the deployment was removed before we got the
			// close timestamp for the endpoint. Use the current timestamp instead.
			if prevTS == timestamp.InfiniteFuture {
				prevTS = timestamp.Now()
			}
			updates = append(updates, ep.ToProto(prevTS))
		}
	}

	return updates
}

// UpdateState updates the internal LastSentState maps with the current state
func (l *LegacyUpdateComputer) UpdateState(currentConns map[*indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[*indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[*indicator.ProcessListening]timestamp.MicroTS) {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	// Update connections state
	l.enrichedConnsLastSentState = make(map[*indicator.NetworkConn]timestamp.MicroTS, len(currentConns))
	for conn, ts := range currentConns {
		l.enrichedConnsLastSentState[conn] = ts
	}

	// Update endpoints state
	l.enrichedEndpointsLastSentState = make(map[*indicator.ContainerEndpoint]timestamp.MicroTS, len(currentEndpoints))
	for ep, ts := range currentEndpoints {
		l.enrichedEndpointsLastSentState[ep] = ts
	}

	// Update processes state
	l.enrichedProcessesLastSentState = make(map[*indicator.ProcessListening]timestamp.MicroTS, len(currentProcesses))
	for proc, ts := range currentProcesses {
		l.enrichedProcessesLastSentState[proc] = ts
	}
}

// ResetState clears all internal LastSentState maps
func (l *LegacyUpdateComputer) ResetState() {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	l.enrichedConnsLastSentState = nil
	l.enrichedEndpointsLastSentState = nil
	l.enrichedProcessesLastSentState = nil
}

// GetStateMetrics returns the size of internal state maps for monitoring
func (l *LegacyUpdateComputer) GetStateMetrics() (connsSize, endpointsSize, processesSize int) {
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()

	return len(l.enrichedConnsLastSentState), len(l.enrichedEndpointsLastSentState), len(l.enrichedProcessesLastSentState)
}

// isUpdated determines whether a connection/endpoint should be included in updates to Central.
// Returns true in the following scenarios:
// 1. New connection/endpoint (!seenPreviously)
// 2. Connection/endpoint timestamp advanced (currTS > prevTS)
// 3. State transition from OPEN -> CLOSED (InfiniteFuture -> actual timestamp)
func isUpdated(prevTS, currTS timestamp.MicroTS, seenPreviously bool) bool {
	// Connection has not been seen in the last tick.
	if !seenPreviously {
		return true
	}
	// Collector saw this connection more recently.
	if currTS > prevTS {
		return true
	}
	// Connection was active (unclosed) in the last tick, now it is closed.
	if prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		return true
	}
	return false
}
