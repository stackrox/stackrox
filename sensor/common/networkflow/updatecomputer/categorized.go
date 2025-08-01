package updatecomputer

import (
	"time"

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

// closedConnEntry stores timestamp information for recently closed connections
type closedConnEntry struct {
	prevTS    timestamp.MicroTS
	expiresAt time.Time
}

// Categorized implements the new categorized update computation logic
// It owns and manages the firstTimeSeen tracking that was previously in the manager
type Categorized struct {
	// State tracking for conditional updates - moved from networkFlowManager
	conditionalUpdatesMutex sync.RWMutex
	firstTimeSeenConns      set.StringSet
	firstTimeSeenEndpoints  set.StringSet
	firstTimeSeenProcesses  set.StringSet

	// Closed connection timestamp tracking for handling late-arriving updates
	closedConnMutex            sync.RWMutex
	closedConnTimestamps       map[string]closedConnEntry
	closedConnRememberDuration time.Duration
	lastCleanup                time.Time
}

// NewCategorized creates a new instance of the categorized update computer
func NewCategorized() *Categorized {
	return &Categorized{
		firstTimeSeenConns:         set.NewStringSet(),
		firstTimeSeenEndpoints:     set.NewStringSet(),
		firstTimeSeenProcesses:     set.NewStringSet(),
		closedConnTimestamps:       make(map[string]closedConnEntry),
		closedConnRememberDuration: env.NetworkFlowClosedConnRememberDuration.DurationSetting(),
		lastCleanup:                time.Now(),
	}
}

func (c *Categorized) ComputeUpdatedConns(current map[*indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	// Perform periodic cleanup first
	c.cleanupExpiredClosedConnections()

	var updates []*storage.NetworkFlow
	var closedConnKeys []string

	// Process current connections using our own categorization
	for conn, currTS := range current {
		connKey := conn.Key()
		isClosed := currTS != timestamp.InfiniteFuture

		// Look up previous timestamp - use infinity for open connections, or actual value for recently closed ones
		found, prevTS := c.lookupPrevTimestamp(connKey, currTS)
		// First, determine the category based on timestamp logic
		category := c.categorizeConnectionUpdate(conn, currTS, prevTS, found)

		switch category {
		case SkipUpdate:
			// Always skip these updates
			continue
		case RequiredUpdate:
			// Always send required updates
			updates = append(updates, conn.ToProto(currTS))
			// If this is a closed connection, store it for future reference
			if isClosed {
				c.storeClosedConnectionTimestamp(connKey, currTS)
				closedConnKeys = append(closedConnKeys, connKey)
			}
		case ConditionalUpdate:
			// Only handle firstTimeSeen logic for conditional updates
			seenPreviously := concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() bool {
				return c.firstTimeSeenConns.Contains(connKey)
			})

			if !isClosed && seenPreviously {
				continue
			}
			// First time seeing this connection - send the update and mark as seen
			concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
				c.firstTimeSeenConns.Add(connKey)
			})
			updates = append(updates, conn.ToProto(currTS))
		}
	}

	// Clean up tracking for closed connections
	if len(closedConnKeys) > 0 {
		c.cleanupConditionalUpdateTracking(closedConnKeys, nil, nil)
	}

	return updates
}

func (c *Categorized) ComputeUpdatedEndpoints(current map[*indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
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

func (c *Categorized) ComputeUpdatedProcesses(current map[*indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
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
func (c *Categorized) UpdateState(currentConns map[*indicator.NetworkConn]timestamp.MicroTS, currentEndpoints map[*indicator.ContainerEndpoint]timestamp.MicroTS, currentProcesses map[*indicator.ProcessListening]timestamp.MicroTS) {
	// No-op: Categorized implementation uses manager's firstTimeSeen tracking
	// State is managed automatically by the categorization functions
}

// ResetState clears the categorized computer's firstTimeSeen tracking
func (c *Categorized) ResetState() {
	c.conditionalUpdatesMutex.Lock()
	defer c.conditionalUpdatesMutex.Unlock()

	// Clear the firstTimeSeen tracking - now owned by this implementation
	c.firstTimeSeenConns = set.NewStringSet()
	c.firstTimeSeenEndpoints = set.NewStringSet()
	c.firstTimeSeenProcesses = set.NewStringSet()

	// Also clear the closed connection tracking
	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps = make(map[string]closedConnEntry)
		c.lastCleanup = time.Now()
	})
}

// GetStateMetrics returns the size of firstTimeSeen tracking for categorized implementation
func (c *Categorized) GetStateMetrics() (connsSize, endpointsSize, processesSize int) {
	connsSize = concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
		return c.firstTimeSeenConns.Cardinality()
	})
	endpointsSize = concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
		return c.firstTimeSeenEndpoints.Cardinality()
	})
	processesSize = concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
		return c.firstTimeSeenProcesses.Cardinality()
	})

	// Note: We don't include closedConnTimestamps size in the metrics as it's a temporary tracking map
	// If needed, this could be added as a separate metric
	return connsSize, endpointsSize, processesSize
}

// categorizeConnectionUpdate determines the update category for a connection based on current and previous state
func (c *Categorized) categorizeConnectionUpdate(
	conn *indicator.NetworkConn,
	currTS timestamp.MicroTS,
	prevTS timestamp.MicroTS,
	prevTsFound bool,
) UpdateCategory {
	if prevTsFound && prevTS == timestamp.InfiniteFuture && currTS != timestamp.InfiniteFuture {
		// Connection closed (state transition OPEN -> CLOSED)
		return RequiredUpdate
	}
	if !prevTsFound && currTS != timestamp.InfiniteFuture {
		// New connection reported as closed
		return RequiredUpdate
	}
	if currTS <= prevTS {
		// Older timestamp than what we already processed or no change - skip
		return SkipUpdate
	}

	// Timestamp update for already closed connection
	if currTS > prevTS && currTS != timestamp.InfiniteFuture {
		return RequiredUpdate
	}

	return ConditionalUpdate
}

// categorizeEndpointUpdate determines the update category for an endpoint
func (c *Categorized) categorizeEndpointUpdate(ep *indicator.ContainerEndpoint, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
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
	isTracked := concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() bool {
		return c.firstTimeSeenEndpoints.Contains(epKey)
	})

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
func (c *Categorized) categorizeProcessUpdate(proc *indicator.ProcessListening, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
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
	isTracked := concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() bool {
		return c.firstTimeSeenProcesses.Contains(procKey)
	})

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

// lookupPrevTimestamp retrieves the previous timestamp for a connection
// For open connections, returns timestamp.InfiniteFuture
// For recently closed connections, returns the stored timestamp if available
func (c *Categorized) lookupPrevTimestamp(connKey string, currTS timestamp.MicroTS) (found bool, prevTS timestamp.MicroTS) {
	// For closed connections, check if we have stored previous timestamp
	return concurrency.WithRLock2(&c.closedConnMutex, func() (bool, timestamp.MicroTS) {
		entry, exists := c.closedConnTimestamps[connKey]
		return exists, entry.prevTS
	})
}

// storeClosedConnectionTimestamp stores the timestamp of a closed connection for future reference
func (c *Categorized) storeClosedConnectionTimestamp(connKey string, closedTS timestamp.MicroTS) {
	expiresAt := time.Now().Add(c.closedConnRememberDuration)

	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps[connKey] = closedConnEntry{
			prevTS:    closedTS,
			expiresAt: expiresAt,
		}
	})
}

// cleanupExpiredClosedConnections removes expired entries from the closed connection tracking map
func (c *Categorized) cleanupExpiredClosedConnections() {
	now := time.Now()

	// Only run cleanup every minute to avoid excessive overhead
	concurrency.WithRLock(&c.closedConnMutex, func() {
		if now.Sub(c.lastCleanup) < time.Minute {
			return
		}
	})

	// Perform the cleanup
	concurrency.WithLock(&c.closedConnMutex, func() {
		// Double-check inside the write lock
		if now.Sub(c.lastCleanup) < time.Minute {
			return
		}

		for key, entry := range c.closedConnTimestamps {
			if now.After(entry.expiresAt) {
				delete(c.closedConnTimestamps, key)
			}
		}

		c.lastCleanup = now
	})
}

// cleanupConditionalUpdateTracking removes tracking entries for closed connections/endpoints
func (c *Categorized) cleanupConditionalUpdateTracking(closedConns []string, closedEndpoints []string, closedProcesses []string) {
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
