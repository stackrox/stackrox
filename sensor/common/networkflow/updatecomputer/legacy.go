package updatecomputer

import (
	"maps"
	"slices"
	"time"

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

	// cachedUpdates contains a list of updates to Central that cannot be sent at the given moment.
	cachedUpdatesConn []*storage.NetworkFlow
	cachedUpdatesEp   []*storage.NetworkEndpoint

	// Mutex to protect the LastSentState maps
	lastSentStateMutex sync.RWMutex
}

// NewLegacy creates a new instance of the legacy update computer
func NewLegacy() *Legacy {
	return &Legacy{
		enrichedConnsLastSentState:     make(map[indicator.NetworkConn]timestamp.MicroTS),
		enrichedEndpointsLastSentState: make(map[indicator.ContainerEndpoint]timestamp.MicroTS),
		enrichedProcessesLastSentState: make(map[indicator.ProcessListening]timestamp.MicroTS),
		cachedUpdatesConn:              make([]*storage.NetworkFlow, 0),
		cachedUpdatesEp:                make([]*storage.NetworkEndpoint, 0),
	}
}

func (l *Legacy) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	if len(current) == 0 {
		// Received an empty map with current state.
		// Return the cache as it may contain past updates collected during the offline mode.
		return l.cachedUpdatesConn
	}
	updates := concurrency.WithRLock1(&l.lastSentStateMutex, func() []*storage.NetworkFlow {
		return computeUpdates(current, l.enrichedConnsLastSentState, func(conn indicator.NetworkConn, ts timestamp.MicroTS) *storage.NetworkFlow {
			return (&conn).ToProto(ts)
		})
	})
	// Store into cache in case sending to Central fails.
	l.cachedUpdatesConn = slices.Grow(l.cachedUpdatesConn, len(updates))
	l.cachedUpdatesConn = append(l.cachedUpdatesConn, updates...)
	// Return concatenated past and current updates.
	return l.cachedUpdatesConn
}

func (l *Legacy) ComputeUpdatedEndpointsAndProcesses(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) ([]*storage.NetworkEndpoint, []*storage.ProcessListeningOnPortFromSensor) {
	if len(enrichedEndpointsProcesses) == 0 {
		// Received an empty map with current state.
		// Return the cache as it may contain past updates collected during the offline mode.
		// Note: We don't cache processes, so return empty slice for those.
		return l.cachedUpdatesEp, []*storage.ProcessListeningOnPortFromSensor{}
	}
	currentEps := make(map[indicator.ContainerEndpoint]timestamp.MicroTS, len(l.enrichedEndpointsLastSentState))
	currentProc := make(map[indicator.ProcessListening]timestamp.MicroTS)
	// Convert the joint map into the legacy format with two maps
	for endpoint, procWithTS := range enrichedEndpointsProcesses {
		currentEps[endpoint] = procWithTS.LastSeen
		if procWithTS.ProcessListening != nil {
			currentProc[*procWithTS.ProcessListening] = procWithTS.LastSeen
		}
	}
	return l.computeUpdatedEndpoints(currentEps), l.computeUpdatedProcesses(currentProc)
}

func (l *Legacy) computeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	epUpdates := concurrency.WithRLock1(&l.lastSentStateMutex, func() []*storage.NetworkEndpoint {
		return computeUpdates(current, l.enrichedEndpointsLastSentState, func(ep indicator.ContainerEndpoint, ts timestamp.MicroTS) *storage.NetworkEndpoint {
			return (&ep).ToProto(ts)
		})
	})
	// Store into cache in case sending to Central fails.
	l.cachedUpdatesEp = slices.Grow(l.cachedUpdatesEp, len(epUpdates))
	l.cachedUpdatesEp = append(l.cachedUpdatesEp, epUpdates...)
	// Return concatenated past and current updates.
	return l.cachedUpdatesEp
}

func (l *Legacy) computeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
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

func (l *Legacy) OnStartSendConnections(currentConns map[indicator.NetworkConn]timestamp.MicroTS) {
	// Clear the cache before sending - the manager now has the items
	l.cachedUpdatesConn = nil

	// Update lastSentState to track what we've seen (for computing future diffs)
	if currentConns != nil {
		l.lastSentStateMutex.Lock()
		defer l.lastSentStateMutex.Unlock()
		l.enrichedConnsLastSentState = maps.Clone(currentConns)
	}
}

func (l *Legacy) OnStartSendEndpoints(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
	// Clear the cache before sending - the manager now has the items
	l.cachedUpdatesEp = nil

	// Update lastSentState to track what we've seen (for computing future diffs)
	if enrichedEndpointsProcesses != nil {
		l.lastSentStateMutex.Lock()
		defer l.lastSentStateMutex.Unlock()
		l.enrichedEndpointsLastSentState = make(map[indicator.ContainerEndpoint]timestamp.MicroTS, len(enrichedEndpointsProcesses))
		for endpoint, procWithTS := range enrichedEndpointsProcesses {
			l.enrichedEndpointsLastSentState[endpoint] = procWithTS.LastSeen
		}
	}
}

func (l *Legacy) OnSendConnectionsFailure(unsentConns []*storage.NetworkFlow) {
	// Store the unsent items in cache for retry
	l.cachedUpdatesConn = unsentConns
}

func (l *Legacy) OnSendEndpointsFailure(unsentEps []*storage.NetworkEndpoint) {
	// Store the unsent items in cache for retry
	l.cachedUpdatesEp = unsentEps
}

// OnSuccessfulSendProcesses contains actions that should be executed after successful sending of processesListening updates to Central.
func (l *Legacy) OnSuccessfulSendProcesses(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
	if enrichedEndpointsProcesses != nil {
		l.lastSentStateMutex.Lock()
		defer l.lastSentStateMutex.Unlock()
		l.enrichedProcessesLastSentState = make(map[indicator.ProcessListening]timestamp.MicroTS, len(enrichedEndpointsProcesses))
		for _, procWithTS := range enrichedEndpointsProcesses {
			if procWithTS.ProcessListening != nil {
				l.enrichedProcessesLastSentState[*procWithTS.ProcessListening] = procWithTS.LastSeen
			}
		}
	}
}

func (l *Legacy) PeriodicCleanup(_ time.Time, _ time.Duration) {}

// ResetState clears all internal LastSentState maps
func (l *Legacy) ResetState() {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	l.enrichedConnsLastSentState = nil
	l.enrichedEndpointsLastSentState = nil
	l.enrichedProcessesLastSentState = nil
	l.cachedUpdatesConn = nil
	l.cachedUpdatesEp = nil
}

func (l *Legacy) RecordSizeMetrics(lenSize, byteSize *prometheus.GaugeVec) {
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
	lenSize.WithLabelValues("lastSent", "processes").Set(float64(lenProc))

	// Avg. byte-size of single element including go map overhead.
	// Estimated with by creating a map with 100k elements, measuring memory consumption (including map overhead)
	// and dividing again by 100k.
	connsSize := 480 * lenConn
	epSize := 330 * lenEp
	procSize := 406 * lenProc
	byteSize.WithLabelValues("lastSent", "conns").Set(float64(connsSize))
	byteSize.WithLabelValues("lastSent", "endpoints").Set(float64(epSize))
	byteSize.WithLabelValues("lastSent", "processes").Set(float64(procSize))

	// Size of buffers that hold updates to Central while Sensor is offline
	lenSize.WithLabelValues("cachedUpdates", string(ConnectionEnrichedEntity)).Set(float64(len(l.cachedUpdatesConn)))
	lenSize.WithLabelValues("cachedUpdates", string(EndpointEnrichedEntity)).Set(float64(len(l.cachedUpdatesEp)))
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
			// This closes all connections that were opened in the last tick and disappeared in the current tick.
			// This literally forces sensor to remember all open connections until they are closed.
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
