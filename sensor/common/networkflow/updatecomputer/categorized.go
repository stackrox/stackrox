package updatecomputer

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
	"github.com/stackrox/rox/sensor/common/networkflow/manager/indicator"
)

// closedConnEntry stores timestamp information for recently closed connections
type closedConnEntry struct {
	prevTS    timestamp.MicroTS
	expiresAt timestamp.MicroTS
}

type EnrichedEntity string

var (
	ConnectionEnrichedEntity     EnrichedEntity = "connection"
	ConnectionHashEnrichedEntity EnrichedEntity = "connection-hash"
	EndpointEnrichedEntity       EnrichedEntity = "endpoint"
	EndpointHashEnrichedEntity   EnrichedEntity = "endpoint-hash"
	ProcessEnrichedEntity        EnrichedEntity = "process"
	ProcessHashEnrichedEntity    EnrichedEntity = "process-hash"
)

var allEntities = []EnrichedEntity{
	ConnectionEnrichedEntity, ConnectionHashEnrichedEntity,
	EndpointEnrichedEntity, EndpointHashEnrichedEntity,
	ProcessEnrichedEntity, ProcessHashEnrichedEntity}

// Categorized implements the new categorized update computation logic
// It owns and manages the firstTimeSeen tracking that was previously in the manager
type Categorized struct {
	// State tracking for conditional updates - moved from networkFlowManager
	openTrackerMutex    sync.RWMutex
	openTracker         map[EnrichedEntity]set.StringSet
	openTrackerEstBytes map[EnrichedEntity]uintptr

	// Closed connection timestamp tracking for handling late-arriving updates
	closedConnMutex            sync.RWMutex
	closedConnTimestamps       map[string]closedConnEntry
	closedConnRememberDuration time.Duration

	lastCleanupMutex sync.RWMutex
	lastCleanup      time.Time

	// Closed endpoint timestamp tracking for handling late-arriving updates
	// closedEndpointMutex      sync.RWMutex
	// closedEndpointTimestamps map[string]closedConnEntry

	// Closed process timestamp tracking for handling late-arriving updates
	// closedProcessMutex      sync.RWMutex
	// closedProcessTimestamps map[string]closedConnEntry
}

// NewCategorized creates a new instance of the categorized update computer
func NewCategorized() *Categorized {
	return &Categorized{
		openTracker: map[EnrichedEntity]set.StringSet{
			ConnectionEnrichedEntity:     set.NewStringSet(),
			ConnectionHashEnrichedEntity: set.NewStringSet(),
			EndpointEnrichedEntity:       set.NewStringSet(),
			ProcessEnrichedEntity:        set.NewStringSet(),
		},
		openTrackerEstBytes: map[EnrichedEntity]uintptr{
			ConnectionEnrichedEntity:     0,
			ConnectionHashEnrichedEntity: 0,
			EndpointEnrichedEntity:       0,
			ProcessEnrichedEntity:        0,
		},
		closedConnTimestamps:       make(map[string]closedConnEntry),
		closedConnRememberDuration: env.NetworkFlowClosedConnRememberDuration.DurationSetting(),
		lastCleanup:                time.Now(),
	}
}

// ComputeUpdatedConns returns a list of updates meant to be sent to Central.
// An error is returned on any anomalous behavior and does not invalidate the results, thus should be treated as warning.
func (c *Categorized) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	var updates []*storage.NetworkFlow
	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates
	}

	// Process currentState connections using our own categorization
	for conn, currTS := range current {
		connKey := conn.Key()
		connKeyHashed := conn.HashedKey()

		// Look up previous timestamp - use infinity for open connections, or actual value for recently closed ones
		prevTsFound, prevTS := c.lookupPrevTimestamp(connKey)
		// Based on the categorization, apply direct action. Execute expensive checks only if necessary.
		// Run only for comparison of memory consumption
		_ = shallUpdate(prevTS, currTS, prevTsFound, connKeyHashed,
			ConnectionHashEnrichedEntity, c.openTracker, &c.openTrackerMutex)

		if shallUpdate(prevTS, currTS, prevTsFound, connKey,
			ConnectionEnrichedEntity, c.openTracker, &c.openTrackerMutex) {
			c.storeClosedConnectionTimestamp(connKey, currTS, c.closedConnRememberDuration)
			updates = append(updates, conn.ToProto(currTS))
		}
	}
	return updates
}

// shallUpdate decides whether an update to Central should be sent for a given enrichment update.
// The function is optimized to execute the lighter checks first and
// for easier reading (some conditions could have been compacted).
func shallUpdate(
	prevTS, currTS timestamp.MicroTS, prevTsFound bool,
	connKey string, ee EnrichedEntity,
	openTracker map[EnrichedEntity]set.StringSet, mutex *sync.RWMutex) bool {

	// Variables for ease of reading
	isClosed := currTS != timestamp.InfiniteFuture
	isOpen := !isClosed
	wasClosed := prevTsFound && prevTS != timestamp.InfiniteFuture

	// CLOSED -> CLOSED
	if wasClosed && isClosed {
		// Update only if currTS is later than prevTS.
		if prevTS < currTS {
			UpdateEvents.WithLabelValues("closed_closed", string(ee), "update").Inc()
			UpdateEventsGauge.WithLabelValues("closed_closed", string(ee), "update").Inc()
			return true
		}
		UpdateEvents.WithLabelValues("closed_closed", string(ee), "skip").Inc()
		UpdateEventsGauge.WithLabelValues("closed_closed", string(ee), "skip").Inc()
		return false
	}
	// CLOSED -> OPEN
	if wasClosed {
		// Track open connection
		concurrency.WithLock(mutex, func() {
			stringSet := openTracker[ee]
			stringSet.Add(connKey)
			openTracker[ee] = stringSet
		})
		UpdateEvents.WithLabelValues("closed_open", string(ee), "update").Inc()
		UpdateEventsGauge.WithLabelValues("closed_open", string(ee), "update").Inc()
		return true
	}
	// OPEN -> OPEN
	if isOpen {
		seenPreviouslyOpen := concurrency.WithRLock1(mutex, func() bool {
			stringSet := openTracker[ee]
			return stringSet.Contains(connKey)
		})
		if seenPreviouslyOpen {
			UpdateEvents.WithLabelValues("open_open", string(ee), "skip_already_seen").Inc()
			UpdateEventsGauge.WithLabelValues("open_open", string(ee), "skip_already_seen").Inc()
			return false
		}
		// Seeing it for the first time.
		concurrency.WithLock(mutex, func() {
			stringSet := openTracker[ee]
			stringSet.Add(connKey)
			openTracker[ee] = stringSet
		})
		UpdateEvents.WithLabelValues("open_open", string(ee), "update").Inc()
		UpdateEventsGauge.WithLabelValues("open_open", string(ee), "update").Inc()
		return true
	}
	// OPEN -> CLOSED
	concurrency.WithLock(mutex, func() {
		stringSet := openTracker[ee]
		stringSet.Remove(connKey)
		openTracker[ee] = stringSet
	})
	UpdateEvents.WithLabelValues("open_closed", string(ee), "update").Inc()
	UpdateEventsGauge.WithLabelValues("open_closed", string(ee), "update").Inc()
	return true
}

// shallUpdateNoPast decides whether an update to Central should be sent for a given enrichment update.
// The function is optimized to execute the lighter checks first and
// for easier reading (some conditions could have been compacted).
// shallUpdateNoPast does not consider the state in previous tick, it only makes a decision based on the data from
// the current tick.
func shallUpdateNoPast(
	currTS timestamp.MicroTS,
	connKey string, ee EnrichedEntity,
	openTracker map[EnrichedEntity]set.StringSet, mutex *sync.RWMutex) bool {
	isClosed := currTS != timestamp.InfiniteFuture

	// UNKNOWN -> CLOSED
	if isClosed {
		// Remove from the open tracker in case this was open in the past
		concurrency.WithLock(mutex, func() {
			stringSet := openTracker[ee]
			stringSet.Remove(connKey)
			openTracker[ee] = stringSet
		})
		UpdateEvents.WithLabelValues("closed_closed", string(ee), "update").Inc()
		UpdateEventsGauge.WithLabelValues("closed_closed", string(ee), "update").Inc()
		return true
	}
	// UNKNOWN -> OPEN
	seenPreviouslyOpen := concurrency.WithRLock1(mutex, func() bool {
		stringSet := openTracker[ee]
		return stringSet.Contains(connKey)
	})
	if seenPreviouslyOpen {
		UpdateEvents.WithLabelValues("open_open", string(ee), "skip_already_seen").Inc()
		UpdateEventsGauge.WithLabelValues("open_open", string(ee), "skip_already_seen").Inc()
		return false
	}
	// Seeing it for the first time.
	concurrency.WithLock(mutex, func() {
		stringSet := openTracker[ee]
		stringSet.Add(connKey)
		openTracker[ee] = stringSet
	})
	UpdateEvents.WithLabelValues("open_open", string(ee), "update").Inc()
	UpdateEventsGauge.WithLabelValues("open_open", string(ee), "update").Inc()
	return true
}

func (c *Categorized) ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	var updates []*storage.NetworkEndpoint

	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates
	}

	// Process current endpoints using our own categorization
	for ep, currTS := range current {
		epKey := ep.Key()
		epHashedKey := ep.HashedKey()
		// Based on the categorization, apply direct action. Execute expensive checks only if necessary.
		_ = shallUpdateNoPast(currTS, epHashedKey,
			EndpointHashEnrichedEntity, c.openTracker, &c.openTrackerMutex)

		if shallUpdateNoPast(currTS, epKey,
			EndpointEnrichedEntity, c.openTracker, &c.openTrackerMutex) {
			updates = append(updates, ep.ToProto(currTS))
		}
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

	if len(current) == 0 {
		// Received an empty map with current state. This may mean the following:
		// - We discarded some items during the enrichment process, so that 0 have made it through.
		// - We run this command on an empty map.
		return updates
	}

	// Process current processes using our own categorization
	for proc, currTS := range current {
		procKey := proc.HashedKey()
		// Based on the categorization, apply direct action. Execute expensive checks only if necessary.
		if shallUpdateNoPast(currTS, procKey,
			ProcessHashEnrichedEntity, c.openTracker, &c.openTrackerMutex) {
			updates = append(updates, proc.ToProto(currTS))
		}
	}

	return updates
}

// UpdateState for categorized implementation triggers update computation as it is impossible to set the internal state otherwise.
func (c *Categorized) UpdateState(currentConns map[indicator.NetworkConn]timestamp.MicroTS,
	currentEndpoints map[indicator.ContainerEndpoint]timestamp.MicroTS,
	currentProcesses map[indicator.ProcessListening]timestamp.MicroTS,
) {
	// Updating state per-se is impossible, but one can trigger a single computation to update the internal state.
	_ = c.ComputeUpdatedConns(currentConns)
	_ = c.ComputeUpdatedEndpoints(currentEndpoints)
	_ = c.ComputeUpdatedProcesses(currentProcesses)
}

func (c *Categorized) updateLastCleanup(now time.Time) {
	c.lastCleanupMutex.Lock()
	defer c.lastCleanupMutex.Unlock()
	c.lastCleanup = now
}

// ResetState clears the categorized computer's firstTimeSeen tracking
func (c *Categorized) ResetState() {
	concurrency.WithLock(&c.openTrackerMutex, func() {
		c.openTracker = map[EnrichedEntity]set.StringSet{
			ConnectionEnrichedEntity:     set.NewStringSet(),
			ConnectionHashEnrichedEntity: set.NewStringSet(),
			EndpointEnrichedEntity:       set.NewStringSet(),
			ProcessEnrichedEntity:        set.NewStringSet(),
		}
		c.openTrackerEstBytes = map[EnrichedEntity]uintptr{
			ConnectionEnrichedEntity:     0,
			ConnectionHashEnrichedEntity: 0,
			EndpointEnrichedEntity:       0,
			ProcessEnrichedEntity:        0,
		}
	})

	c.updateLastCleanup(time.Now())

	// Also clear the closed connection tracking
	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps = make(map[string]closedConnEntry)
	})
}

func (c *Categorized) RecordSizeMetrics(name string, lenSize, byteSize *prometheus.GaugeVec) {
	for _, entity := range allEntities {
		value := concurrency.WithRLock1(&c.openTrackerMutex, func() int {
			return c.openTracker[entity].Cardinality()
		})
		lenSize.WithLabelValues("open", string(entity)).Set(float64(value))
	}
	value := concurrency.WithRLock1(&c.closedConnMutex, func() int {
		return len(c.closedConnTimestamps)
	})
	lenSize.WithLabelValues("closedTimestamps", string(ConnectionEnrichedEntity)).Set(float64(value))

	// Calculate byte metrics
	for _, entity := range allEntities {
		baseSize := concurrency.WithRLock1(&c.openTrackerMutex, func() uintptr {
			var totalStringBytes uintptr
			for _, s := range c.openTracker[entity].AsSlice() {
				totalStringBytes += uintptr(len(s))
			}
			return 8 + uintptr(c.openTracker[entity].Cardinality())*16 + totalStringBytes
		})
		c.openTrackerEstBytes[entity] = baseSize * 2 // *2 comes from the overhead for map
		byteSize.WithLabelValues("open", string(entity)).Set(float64(c.openTrackerEstBytes[entity]))
	}
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
func (c *Categorized) storeClosedConnectionTimestamp(
	connKey string, closedTS timestamp.MicroTS, closedConnRememberDuration time.Duration) {
	// Do not store open connections.
	if closedTS == timestamp.InfiniteFuture {
		return
	}
	expiresAt := closedTS.Add(closedConnRememberDuration)

	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps[connKey] = closedConnEntry{
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
			if timestamp.FromGoTime(now).After(entry.expiresAt) {
				delete(c.closedConnTimestamps, key)
			}
		}
	})
}
