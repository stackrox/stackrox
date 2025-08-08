package updatecomputer

import (
	"time"

	"github.com/pkg/errors"
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

	lastCleanupMutex sync.RWMutex
	lastCleanup      time.Time

	// Closed endpoint timestamp tracking for handling late-arriving updates
	closedEndpointMutex      sync.RWMutex
	closedEndpointTimestamps map[string]closedConnEntry

	// Closed process timestamp tracking for handling late-arriving updates
	closedProcessMutex      sync.RWMutex
	closedProcessTimestamps map[string]closedConnEntry
}

// NewCategorized creates a new instance of the categorized update computer
func NewCategorized() *Categorized {
	return &Categorized{
		firstTimeSeenConns:         set.NewStringSet(),
		firstTimeSeenEndpoints:     set.NewStringSet(),
		firstTimeSeenProcesses:     set.NewStringSet(),
		closedConnTimestamps:       make(map[string]closedConnEntry),
		closedEndpointTimestamps:   make(map[string]closedConnEntry),
		closedProcessTimestamps:    make(map[string]closedConnEntry),
		closedConnRememberDuration: env.NetworkFlowClosedConnRememberDuration.DurationSetting(),
		lastCleanup:                time.Now(),
	}
}

// ComputeUpdatedConns returns a list of updates meant to be sent to Central.
// An error is returned on any anomalous behavior and does not invalidate the results, thus should be treated as warning.
func (c *Categorized) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) ([]*storage.NetworkFlow, error) {
	var updates []*storage.NetworkFlow
	var closedEntities []string

	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates, errors.New("no updates available")
	}

	// Process currentState connections using our own categorization
	for conn, currTS := range current {
		connKey := conn.Key()
		isClosed := currTS != timestamp.InfiniteFuture

		// Look up previous timestamp - use infinity for open connections, or actual value for recently closed ones
		found, prevTS := c.lookupPrevTimestamp(connKey)
		// First, determine the category based on timestamp logic
		category := c.categorizeUpdate(currTS, prevTS, found)

		switch category {
		case SkipUpdate:
			// Always skip these updates
			continue
		case RequiredUpdate:
			// Always send required updates
			updates = append(updates, conn.ToProto(currTS))
			// If this is a closed connection, store it for future reference
			if isClosed {
				c.storeClosedConnectionTimestamp(connKey, currTS, c.closedConnRememberDuration)
				closedEntities = append(closedEntities, connKey)
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
	if len(closedEntities) > 0 {
		c.cleanupConditionalUpdateTracking(closedEntities, nil, nil)
	}

	return updates, nil
}

func (c *Categorized) ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	var updates []*storage.NetworkEndpoint
	var closedEntities []string

	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates
	}

	// Process current endpoints using our own categorization
	for ep, currTS := range current {
		epKey := ep.Key()
		isClosed := currTS != timestamp.InfiniteFuture

		// Look up previous timestamp - use infinity for open endpoints, or actual value for recently closed ones
		found, prevTS := c.lookupPrevEndpointTimestamp(epKey)
		// First, determine the category based on timestamp logic
		category := c.categorizeUpdate(currTS, prevTS, found)

		switch category {
		case SkipUpdate:
			// Always skip these updates
			continue
		case RequiredUpdate:
			// Always send required updates
			updates = append(updates, ep.ToProto(currTS))
			// If this is a closed endpoint, store it for future reference
			if isClosed {
				c.storeClosedEndpointTimestamp(epKey, currTS, c.closedConnRememberDuration)
				closedEntities = append(closedEntities, epKey)
			}
		case ConditionalUpdate:
			// Only handle firstTimeSeen logic for conditional updates
			seenPreviously := concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() bool {
				return c.firstTimeSeenEndpoints.Contains(epKey)
			})

			if !isClosed && seenPreviously {
				continue
			}
			// First time seeing this endpoint - send the update and mark as seen
			concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
				c.firstTimeSeenEndpoints.Add(epKey)
			})
			updates = append(updates, ep.ToProto(currTS))
		}
	}

	// Clean up tracking for closed endpoints
	if len(closedEntities) > 0 {
		c.cleanupConditionalUpdateTracking(nil, closedEntities, nil)
	}

	return updates
}

func (c *Categorized) ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().WarnL(loggingRateLimiter,
				"Received process(es) while ProcessesListeningOnPort feature is disabled. This may indicate a misconfiguration.")
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}

	var updates []*storage.ProcessListeningOnPortFromSensor
	var closedEntities []string

	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates
	}

	// Process current processes using our own categorization
	for proc, currTS := range current {
		procKey := proc.Key()
		isClosed := currTS != timestamp.InfiniteFuture

		// Look up previous timestamp - use infinity for open processes, or actual value for recently closed ones
		found, prevTS := c.lookupPrevProcessTimestamp(procKey)
		// First, determine the category based on timestamp logic
		category := c.categorizeUpdate(currTS, prevTS, found)

		switch category {
		case SkipUpdate:
			// Always skip these updates
			continue
		case RequiredUpdate:
			// Always send required updates
			updates = append(updates, proc.ToProto(currTS))
			// If this is a closed process, store it for future reference
			if isClosed {
				c.storeClosedProcessTimestamp(procKey, currTS, c.closedConnRememberDuration)
				closedEntities = append(closedEntities, procKey)
			}
		case ConditionalUpdate:
			// Only handle firstTimeSeen logic for conditional updates
			seenPreviously := concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() bool {
				return c.firstTimeSeenProcesses.Contains(procKey)
			})

			if !isClosed && seenPreviously {
				continue
			}
			// First time seeing this process - send the update and mark as seen
			concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
				c.firstTimeSeenProcesses.Add(procKey)
			})
			updates = append(updates, proc.ToProto(currTS))
		}
	}

	// Clean up tracking for closed processes
	if len(closedEntities) > 0 {
		c.cleanupConditionalUpdateTracking(nil, nil, closedEntities)
	}

	return updates
}

// UpdateState for categorized implementation is a no-op since it uses firstTimeSeen tracking
func (c *Categorized) UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS,
	currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS,
	currentProcesses map[indicator.ProcessListening]timestamp.MicroTS,
) {
	// Updating state per-se is impossible, but one can trigger a single computation to update the internal state.
	_, _ = c.ComputeUpdatedConns(currentConns)
	c.ComputeUpdatedEndpoints(currentEndpoints)
	c.ComputeUpdatedProcesses(currentProcesses)
}

func (c *Categorized) updateLastCleanup(now time.Time) {
	c.lastCleanupMutex.Lock()
	defer c.lastCleanupMutex.Unlock()
	c.lastCleanup = now
}

// ResetState clears the categorized computer's firstTimeSeen tracking
func (c *Categorized) ResetState() {
	concurrency.WithLock(&c.conditionalUpdatesMutex, func() {
		c.firstTimeSeenConns = set.NewStringSet()
		c.firstTimeSeenEndpoints = set.NewStringSet()
		c.firstTimeSeenProcesses = set.NewStringSet()
	})

	c.updateLastCleanup(time.Now())

	// Also clear the closed connection tracking
	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps = make(map[string]closedConnEntry)
	})

	// Clear the closed endpoint tracking
	concurrency.WithLock(&c.closedEndpointMutex, func() {
		c.closedEndpointTimestamps = make(map[string]closedConnEntry)
	})

	// Clear the closed process tracking
	concurrency.WithLock(&c.closedProcessMutex, func() {
		c.closedProcessTimestamps = make(map[string]closedConnEntry)
	})
}

// GetStateMetrics returns the size of firstTimeSeen tracking for categorized implementation
func (c *Categorized) GetStateMetrics() map[string]map[string]int {
	data := make(map[string]map[string]int)
	data["firstTimeSeen"] = map[string]int{
		"connections": concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
			return c.firstTimeSeenConns.Cardinality()
		}),
		"endpoints": concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
			return c.firstTimeSeenEndpoints.Cardinality()
		}),
		"processes": concurrency.WithRLock1(&c.conditionalUpdatesMutex, func() int {
			return c.firstTimeSeenProcesses.Cardinality()
		}),
	}
	data["closedTimestamps"] = map[string]int{
		"connections": concurrency.WithRLock1(&c.closedConnMutex, func() int {
			return len(c.closedConnTimestamps)
		}),
		"endpoints": concurrency.WithRLock1(&c.closedEndpointMutex, func() int {
			return len(c.closedEndpointTimestamps)
		}),
		"processes": concurrency.WithRLock1(&c.closedProcessMutex, func() int {
			return len(c.closedProcessTimestamps)
		}),
	}
	return data
}

// categorizeUpdate determines the update category for a connection based on currentState and previous state
func (c *Categorized) categorizeUpdate(
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

// lookupPrevTimestamp retrieves the previous timestamp for a connection
// For open connections, returns timestamp.InfiniteFuture
// For recently closed connections, returns the stored timestamp if available
func (c *Categorized) lookupPrevTimestamp(connKey string) (found bool, prevTS timestamp.MicroTS) {
	// For closed connections, check if we have stored previous timestamp
	c.closedConnMutex.RLock()
	defer c.closedConnMutex.RUnlock()
	entry, exists := c.closedConnTimestamps[connKey]
	return exists, entry.prevTS
}

// storeClosedConnectionTimestamp stores the timestamp of a closed connection for future reference
func (c *Categorized) storeClosedConnectionTimestamp(connKey string, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration) {
	expiresAt := time.Now().Add(closedConnRememberDuration)

	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps[connKey] = closedConnEntry{
			prevTS:    closedTS,
			expiresAt: expiresAt,
		}
	})
}

// lookupPrevEndpointTimestamp retrieves the previous timestamp for an endpoint
// For open endpoints, returns timestamp.InfiniteFuture
// For recently closed endpoints, returns the stored timestamp if available
func (c *Categorized) lookupPrevEndpointTimestamp(endpointKey string) (found bool, prevTS timestamp.MicroTS) {
	// For closed endpoints, check if we have stored previous timestamp
	c.closedEndpointMutex.RLock()
	defer c.closedEndpointMutex.RUnlock()
	entry, exists := c.closedEndpointTimestamps[endpointKey]
	return exists, entry.prevTS
}

// storeClosedEndpointTimestamp stores the timestamp of a closed endpoint for future reference
func (c *Categorized) storeClosedEndpointTimestamp(endpointKey string, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration) {
	expiresAt := time.Now().Add(closedConnRememberDuration)

	concurrency.WithLock(&c.closedEndpointMutex, func() {
		c.closedEndpointTimestamps[endpointKey] = closedConnEntry{
			prevTS:    closedTS,
			expiresAt: expiresAt,
		}
	})
}

// lookupPrevProcessTimestamp retrieves the previous timestamp for a process
// For open processes, returns timestamp.InfiniteFuture
// For recently closed processes, returns the stored timestamp if available
func (c *Categorized) lookupPrevProcessTimestamp(processKey string) (found bool, prevTS timestamp.MicroTS) {
	// For closed processes, check if we have stored previous timestamp
	c.closedProcessMutex.RLock()
	defer c.closedProcessMutex.RUnlock()
	entry, exists := c.closedProcessTimestamps[processKey]
	return exists, entry.prevTS
}

// storeClosedProcessTimestamp stores the timestamp of a closed process for future reference
func (c *Categorized) storeClosedProcessTimestamp(processKey string, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration) {
	expiresAt := time.Now().Add(closedConnRememberDuration)

	concurrency.WithLock(&c.closedProcessMutex, func() {
		c.closedProcessTimestamps[processKey] = closedConnEntry{
			prevTS:    closedTS,
			expiresAt: expiresAt,
		}
	})
}

func (c *Categorized) PeriodicCleanup(now time.Time, cleanupInterval time.Duration) {
	// Only run cleanup every minute to avoid excessive overhead
	concurrency.WithRLock(&c.lastCleanupMutex, func() {
		if now.Sub(c.lastCleanup) < cleanupInterval {
			return
		}
	})

	defer c.updateLastCleanup(now)

	// Perform the cleanup
	concurrency.WithLock(&c.closedConnMutex, func() {
		for key, entry := range c.closedConnTimestamps {
			if now.After(entry.expiresAt) {
				delete(c.closedConnTimestamps, key)
			}
		}
	})

	concurrency.WithLock(&c.closedEndpointMutex, func() {
		for key, entry := range c.closedEndpointTimestamps {
			if now.After(entry.expiresAt) {
				delete(c.closedEndpointTimestamps, key)
			}
		}
	})

	concurrency.WithLock(&c.closedProcessMutex, func() {
		for key, entry := range c.closedProcessTimestamps {
			if now.After(entry.expiresAt) {
				delete(c.closedProcessTimestamps, key)
			}
		}
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
