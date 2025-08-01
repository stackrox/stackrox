package updatecomputer

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

var log = logging.LoggerForModule()

// UpdateCategory represents the categorization of network flow updates for sending to Central
type UpdateCategory int

const (
	// RequiredUpdate - Must send to Central (e.g., closing connections, new connections)
	RequiredUpdate UpdateCategory = iota
	// ConditionalUpdate - May send to Central (e.g., first update for open connections)
	ConditionalUpdate
	// SkipUpdate - Don't send to Central (e.g., duplicates, older timestamps)
	SkipUpdate
)

// CategorizedUpdateComputer implements the new categorized update computation logic
// It owns and manages the firstTimeSeen tracking that was previously in the manager
type CategorizedUpdateComputer struct {
	// State tracking for conditional updates - moved from networkFlowManager
	conditionalUpdatesMutex sync.RWMutex
	firstTimeSeenConns      set.StringSet
	firstTimeSeenEndpoints  set.StringSet
	firstTimeSeenProcesses  set.StringSet
}

// NewCategorizedUpdateComputer creates a new instance of the categorized update computer
func NewCategorizedUpdateComputer() UpdateComputer {
	return &CategorizedUpdateComputer{
		firstTimeSeenConns:     set.NewStringSet(),
		firstTimeSeenEndpoints: set.NewStringSet(),
		firstTimeSeenProcesses: set.NewStringSet(),
	}
}

func (c *CategorizedUpdateComputer) ComputeUpdatedConns(current map[*indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	// Use the categorized computer's own categorization logic and firstTimeSeen tracking
	var updates []*storage.NetworkFlow
	var closedConnKeys []string

	// Process current connections using our own categorization
	for conn, currTS := range current {
		// Check if we've seen this connection before using our internal tracking
		connKey := conn.Key()
		var seenPreviously bool
		concurrency.WithRLock(&c.conditionalUpdatesMutex, func() {
			seenPreviously = c.firstTimeSeenConns.Contains(connKey)
		})

		category := c.categorizeConnectionUpdate(conn, currTS, 0, seenPreviously)

		switch category {
		case RequiredUpdate, ConditionalUpdate:
			updates = append(updates, conn.ToProto(currTS))
			// If this is a closed connection, track it for cleanup
			if currTS != timestamp.InfiniteFuture {
				closedConnKeys = append(closedConnKeys, conn.Key())
			}
		case SkipUpdate:
			// Skip this update
		}
	}

	// Clean up tracking for closed connections
	if len(closedConnKeys) > 0 {
		c.cleanupConditionalUpdateTracking(closedConnKeys, nil, nil)
	}

	return updates
}

func (c *CategorizedUpdateComputer) ComputeUpdatedEndpoints(current map[*indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	var updates []*storage.NetworkEndpoint
	var closedEndpointKeys []string

	for ep, currTS := range current {
		category := c.categorizeEndpointUpdate(ep, currTS, 0, false)

		switch category {
		case RequiredUpdate, ConditionalUpdate:
			updates = append(updates, ep.ToProto(currTS))
			if currTS != timestamp.InfiniteFuture {
				closedEndpointKeys = append(closedEndpointKeys, ep.Key())
			}
		case SkipUpdate:
			// Skip this update
		}
	}

	if len(closedEndpointKeys) > 0 {
		c.cleanupConditionalUpdateTracking(nil, closedEndpointKeys, nil)
	}

	return updates
}

func (c *CategorizedUpdateComputer) ComputeUpdatedProcesses(current map[*indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().Warnf(loggingRateLimiter,
				"Received %d process(es) while ProcessesListeningOnPort feature is disabled. This may indicate a misconfiguration.", len(current))
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}

	var updates []*storage.ProcessListeningOnPortFromSensor
	var closedProcessKeys []string

	for proc, currTS := range current {
		category := c.categorizeProcessUpdate(proc, currTS, 0, false)

		switch category {
		case RequiredUpdate, ConditionalUpdate:
			updates = append(updates, proc.ToProto(currTS))
			if currTS != timestamp.InfiniteFuture {
				closedProcessKeys = append(closedProcessKeys, proc.Key())
			}
		case SkipUpdate:
			// Skip this update
		}
	}

	if len(closedProcessKeys) > 0 {
		c.cleanupConditionalUpdateTracking(nil, nil, closedProcessKeys)
	}

	return updates
}

// UpdateState for categorized implementation is a no-op since it uses firstTimeSeen tracking
func (c *CategorizedUpdateComputer) UpdateState(currentConns map[*indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[*indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[*indicator.ProcessListening]timestamp.MicroTS) {
	// No-op: Categorized implementation uses manager's firstTimeSeen tracking
	// State is managed automatically by the categorization functions
}

// ResetState clears the categorized computer's firstTimeSeen tracking
func (c *CategorizedUpdateComputer) ResetState() {
	c.conditionalUpdatesMutex.Lock()
	defer c.conditionalUpdatesMutex.Unlock()

	// Clear the firstTimeSeen tracking - now owned by this implementation
	c.firstTimeSeenConns = set.NewStringSet()
	c.firstTimeSeenEndpoints = set.NewStringSet()
	c.firstTimeSeenProcesses = set.NewStringSet()
}

// GetStateMetrics returns the size of firstTimeSeen tracking for categorized implementation
func (c *CategorizedUpdateComputer) GetStateMetrics() (connsSize, endpointsSize, processesSize int) {
	c.conditionalUpdatesMutex.RLock()
	defer c.conditionalUpdatesMutex.RUnlock()

	return c.firstTimeSeenConns.Cardinality(), c.firstTimeSeenEndpoints.Cardinality(), c.firstTimeSeenProcesses.Cardinality()
}

// categorizeConnectionUpdate determines the update category for a connection based on current and previous state
func (c *CategorizedUpdateComputer) categorizeConnectionUpdate(conn *indicator.NetworkConn, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
	// Category 1: Required updates that must be sent
	if !seenPreviously {
		// New connection never seen before
		return RequiredUpdate
	}
	if prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		// Connection closed (state transition OPEN -> CLOSED)
		return RequiredUpdate
	}
	if currTS < prevTS {
		// Older timestamp than what we already processed - skip
		return SkipUpdate
	}
	if currTS == prevTS {
		// No change in timestamp - duplicate update
		return SkipUpdate
	}

	// Category 2: Conditional updates - only send if it's the first update for an open connection
	// Check if this is the first time we're seeing this open connection in the current tracking period
	connKey := conn.Key()
	c.conditionalUpdatesMutex.RLock()
	isTracked := c.firstTimeSeenConns.Contains(connKey)
	c.conditionalUpdatesMutex.RUnlock()

	if !isTracked {
		// First time seeing this connection in current tracking period
		concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
			c.firstTimeSeenConns.Add(connKey)
		})
		return ConditionalUpdate
	}

	// We've seen this connection before in the current period
	if currTS > prevTS && currTS != timestamp.InfiniteFuture {
		// Subsequent update for an open connection - skip
		return SkipUpdate
	}

	return ConditionalUpdate
}

// categorizeEndpointUpdate determines the update category for an endpoint
func (c *CategorizedUpdateComputer) categorizeEndpointUpdate(ep *indicator.ContainerEndpoint, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
	// Similar logic to connections
	if !seenPreviously {
		return RequiredUpdate
	}
	if prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		return RequiredUpdate
	}
	if currTS < prevTS {
		return SkipUpdate
	}
	if currTS == prevTS {
		return SkipUpdate
	}

	// Check first-time-seen tracking for endpoints
	epKey := ep.Key()
	c.conditionalUpdatesMutex.RLock()
	isTracked := c.firstTimeSeenEndpoints.Contains(epKey)
	c.conditionalUpdatesMutex.RUnlock()

	if !isTracked {
		concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
			c.firstTimeSeenEndpoints.Add(epKey)
		})
		return ConditionalUpdate
	}

	if currTS > prevTS && currTS != timestamp.InfiniteFuture {
		return SkipUpdate
	}

	return ConditionalUpdate
}

// categorizeProcessUpdate determines the update category for a process
func (c *CategorizedUpdateComputer) categorizeProcessUpdate(proc *indicator.ProcessListening, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
	// Similar logic to connections
	if !seenPreviously {
		return RequiredUpdate
	}
	if prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		return RequiredUpdate
	}
	if currTS < prevTS {
		return SkipUpdate
	}
	if currTS == prevTS {
		return SkipUpdate
	}

	// Check first-time-seen tracking for processes
	procKey := proc.Key()
	c.conditionalUpdatesMutex.RLock()
	isTracked := c.firstTimeSeenProcesses.Contains(procKey)
	c.conditionalUpdatesMutex.RUnlock()

	if !isTracked {
		concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
			c.firstTimeSeenProcesses.Add(procKey)
		})
		return ConditionalUpdate
	}

	if currTS > prevTS && currTS != timestamp.InfiniteFuture {
		return SkipUpdate
	}

	return ConditionalUpdate
}

// cleanupConditionalUpdateTracking removes tracking entries for closed connections/endpoints
func (c *CategorizedUpdateComputer) cleanupConditionalUpdateTracking(closedConns []string, closedEndpoints []string, closedProcesses []string) {
	if len(closedConns)+len(closedEndpoints)+len(closedProcesses) == 0 {
		return
	}

	c.conditionalUpdatesMutex.Lock()
	defer c.conditionalUpdatesMutex.Unlock()

	// Clean up entries for connections we know are closed
	for _, connKey := range closedConns {
		c.firstTimeSeenConns.Remove(connKey)
	}

	// Clean up entries for endpoints we know are closed
	for _, epKey := range closedEndpoints {
		c.firstTimeSeenEndpoints.Remove(epKey)
	}

	// Clean up entries for processes we know are closed
	for _, procKey := range closedProcesses {
		c.firstTimeSeenProcesses.Remove(procKey)
	}

	// Log cleanup metrics
	log.Debugf("Cleaned up conditional update tracking for %d conns, %d endpoints, %d processes. Remaining entries: conns=%d, endpoints=%d, processes=%d",
		len(closedConns), len(closedEndpoints), len(closedProcesses),
		c.firstTimeSeenConns.Cardinality(), c.firstTimeSeenEndpoints.Cardinality(), c.firstTimeSeenProcesses.Cardinality())
}
