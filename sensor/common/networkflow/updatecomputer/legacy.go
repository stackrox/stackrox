package updatecomputer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

const loggingRateLimiter = "plop-feature-disabled"

// Legacy implements the original update computation logic using LastSentState maps
// It owns and manages the LastSentState maps that were previously in the manager
type Legacy struct {
	// State tracking maps - these were previously in networkFlowManager
	enrichedConnsLastSentState     map[indicator.NetworkConn]timestamp.MicroTS
	enrichedEndpointsLastSentState map[indicator.ContainerEndpoint]timestamp.MicroTS
	enrichedProcessesLastSentState map[indicator.ProcessListening]timestamp.MicroTS

	// Mutex to protect the LastSentState maps
	lastSentStateMutex sync.RWMutex
}

// NewLegacy creates a new instance of the legacy update computer
func NewLegacy() *Legacy {
	return &Legacy{
		enrichedConnsLastSentState:     make(map[indicator.NetworkConn]timestamp.MicroTS),
		enrichedEndpointsLastSentState: make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
		enrichedProcessesLastSentState: make(map[indicator.ProcessListening]timestamp.MicroTS),
	}
}

func (l *Legacy) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	return concurrency.WithRLock1(&l.lastSentStateMutex, func() []*storage.NetworkFlow {
		return computeUpdates(current, l.enrichedConnsLastSentState, func(conn indicator.NetworkConn, ts timestamp.MicroTS) *storage.NetworkFlow {
			return (&conn).ToProto(ts)
		})
	})
}

func (l *Legacy) ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	return concurrency.WithRLock1(&l.lastSentStateMutex, func() []*storage.NetworkEndpoint {
		return computeUpdates(current, l.enrichedEndpointsLastSentState, func(ep indicator.ContainerEndpoint, ts timestamp.MicroTS) *storage.NetworkEndpoint {
			return (&ep).ToProto(ts)
		})
	})
}

func (l *Legacy) ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().Warn(loggingRateLimiter,
				"Received process(es) while ProcessesListeningOnPort feature is disabled. This may indicate a misconfiguration.")
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}
	return concurrency.WithRLock1(&l.lastSentStateMutex, func() []*storage.ProcessListeningOnPortFromSensor {
		return computeUpdates(current, l.enrichedProcessesLastSentState, func(proc indicator.ProcessListening, ts timestamp.MicroTS) *storage.ProcessListeningOnPortFromSensor {
			return (&proc).ToProto(ts)
		})
	})
}

// UpdateState updates the internal LastSentState maps with the currentState state
func (l *Legacy) UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS,
	currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS,
	currentProcesses map[indicator.ProcessListening]timestamp.MicroTS,
) {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	// Update connections state
	l.enrichedConnsLastSentState = make(map[indicator.NetworkConn]timestamp.MicroTS, len(currentConns))
	for conn, ts := range currentConns {
		l.enrichedConnsLastSentState[conn] = ts
	}

	// Update endpoints state
	l.enrichedEndpointsLastSentState = make(map[indicator.ContainerEndpoint]timestamp.MicroTS, len(currentEndpoints))
	for ep, ts := range currentEndpoints {
		l.enrichedEndpointsLastSentState[ep] = ts
	}

	// Update processes state
	l.enrichedProcessesLastSentState = make(map[indicator.ProcessListening]timestamp.MicroTS, len(currentProcesses))
	for proc, ts := range currentProcesses {
		l.enrichedProcessesLastSentState[proc] = ts
	}
}

// ResetState clears all internal LastSentState maps
func (l *Legacy) ResetState() {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	l.enrichedConnsLastSentState = nil
	l.enrichedEndpointsLastSentState = nil
	l.enrichedProcessesLastSentState = nil
}

func (l *Legacy) RecordSizeMetrics(name string, lenSize, byteSize *prometheus.GaugeVec) {
	lenConn := concurrency.WithRLock1(&l.lastSentStateMutex, func() int {
		return len(l.enrichedConnsLastSentState)
	})
	lenEp := concurrency.WithRLock1(&l.lastSentStateMutex, func() int {
		return len(l.enrichedEndpointsLastSentState)
	})
	lenProc := concurrency.WithRLock1(&l.lastSentStateMutex, func() int {
		return len(l.enrichedProcessesLastSentState)
	})
	lenSize.WithLabelValues("lastSent", "conns").Set(float64(lenConn))
	lenSize.WithLabelValues("lastSent", "endpoints").Set(float64(lenEp))
	lenSize.WithLabelValues("lastSent", "endpoints").Set(float64(lenProc))

	// Avg. byte-size of single element including go map overhead (estimated with Cursor)
	connsSize := 480 * lenConn
	epSize := 330 * lenEp
	byteSize.WithLabelValues("lastSent", "conns").Set(float64(connsSize))
	byteSize.WithLabelValues("lastSent", "endpoints").Set(float64(epSize))
	// TODO: Processes size
}

// computeUpdates is a generic helper for computing updates using the legacy LastSentState approach
func computeUpdates[K comparable, V any](
	current map[K]timestamp.MicroTS,
	lastSentState map[K]timestamp.MicroTS,
	toProto func(K, timestamp.MicroTS) V,
) []V {
	var updates []V

	// Check currentState items for updates
	for key, currTS := range current {
		prevTS, seenPreviously := lastSentState[key]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, toProto(key, currTS))
		}
	}

	// Check for items that are no longer currentState (removed items)
	for key, prevTS := range lastSentState {
		if _, ok := current[key]; !ok {
			// For removed items, use the previous timestamp or currentState time if it was infinite
			finalTS := prevTS
			if prevTS == timestamp.InfiniteFuture {
				finalTS = timestamp.Now()
			}
			updates = append(updates, toProto(key, finalTS))
		}
	}

	return updates
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
