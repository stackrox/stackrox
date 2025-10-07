package updatecomputer

import (
	"slices"
	"strings"
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

var log = logging.LoggerForModule()

type deduperAction int

const (
	deduperActionAdd deduperAction = iota
	deduperActionRemove
	deduperActionUpdateProcess
	deduperActionNoop

	// skipReason explains why a given update was not sent to Central. Used in metric labels.
	skipReasonTimestampOlder = "timestamp_older"
	skipReasonAlreadySeen    = "already_seen"
	skipReasonNone           = "none"

	// updateAction represents a decision whether to update Central. Used in metric labels.
	updateActionUpdate = "update"
	updateActionSkip   = "skip"
)

// closedConnEntry stores timestamp information for recently closed connections
type closedConnEntry struct {
	prevTS    timestamp.MicroTS
	expiresAt timestamp.MicroTS
}

// EnrichedEntity (abbreviated EE) describes the entity being enriched.
// Currently, it represents one of: connection, endpoint, process.
type EnrichedEntity string

var (
	ConnectionEnrichedEntity EnrichedEntity = "connection"
	EndpointEnrichedEntity   EnrichedEntity = "endpoint"
	ProcessEnrichedEntity    EnrichedEntity = "process"
)

// TransitionType describes the type of transition of states - in the previous tick and in the current tick -
// of an enriched entity (EE).
type TransitionType int

const (
	// TransitionTypeOpen2Open describes the situation when a previously seen open EE is seen again as open.
	TransitionTypeOpen2Open TransitionType = iota
	// TransitionTypeNew2Open describes the situation when a new open EE is seen for the first time.
	TransitionTypeNew2Open
	// TransitionTypeOpen2Closed describes the situation when a previously seen open EE is closed.
	TransitionTypeOpen2Closed
	// TransitionTypeClosed2Closed describes the situation when a previously seen closed EE is seen again as closed.
	TransitionTypeClosed2Closed
	// TransitionTypeClosed2Open describes the situation when a previously seen closed EE is seen again as open.
	TransitionTypeClosed2Open
	TransitionTypeReplaceProcess
)

func (tt *TransitionType) String() string {
	switch *tt {
	case TransitionTypeOpen2Open:
		return "open->open"
	case TransitionTypeNew2Open:
		return "new->open"
	case TransitionTypeOpen2Closed:
		return "open->closed"
	case TransitionTypeClosed2Closed:
		return "closed->closed"
	case TransitionTypeClosed2Open:
		return "closed->open"
	case TransitionTypeReplaceProcess:
		return "replace-process"
	}
	return "unknown"
}

// TransitionBased is an update computer that calculates updates based on the type of state transition for each enriched entity.
// It categorizes state transitions to perform the most basic checks first, saving computational resources.
// For example: handle connections being closed (transitions ANY->Closed) first, since there's no need to check whether
// a connection was previously seen (closing a connection almost always requires sending an update).
//
// The main advantage of TransitionBased is that it doesn't need to remember the updates sent to Central
// in the previous tick. Instead, it must remember all open connections that haven't been closed yet (a disadvantage),
// but this can be done in a memory-efficient way by storing only a fingerprint of each connection in a deduper.
//
// It remembers recently closed connections (but not endpoints and processes) for a duration bound to the afterglow period
// to avoid sending duplicate close updates to Central. In the future, after careful investigation,
// this behavior may be made optional and hidden behind an environment variable.
type TransitionBased struct {
	// Algorithm used for creating fingerprints of indicators
	hashingAlgo indicator.HashingAlgo

	// State tracking for conditional updates - moved from networkFlowManager
	connectionsDeduperMutex sync.RWMutex
	connectionsDeduper      *set.StringSet

	endpointsDeduperMutex sync.RWMutex
	// TODO(ROX-31012): Save even more memory by changing to `type b8 [8]byte; map[b8]b8
	endpointsDeduper map[string]string

	// cachedUpdates contains a list of updates to Central that cannot be sent at the given moment.
	cachedUpdatesConn []*storage.NetworkFlow
	cachedUpdatesEp   []*storage.NetworkEndpoint
	cachedUpdatesProc []*storage.ProcessListeningOnPortFromSensor

	// Closed connection timestamp tracking for handling late-arriving updates
	closedConnMutex            sync.RWMutex
	closedConnTimestamps       map[string]closedConnEntry
	closedConnRememberDuration time.Duration

	lastCleanupMutex sync.RWMutex
	lastCleanup      time.Time
}

// newStringSetPtr returns a pointer to a new string set, which is originally a value type.
// This avoids copying the set when it is used in the deduper.
func newStringSetPtr() *set.StringSet {
	s := set.NewStringSet()
	return &s
}

func hashingAlgoFromEnv(v env.Setting) indicator.HashingAlgo {
	switch strings.ToLower(v.Setting()) {
	case "fnv64":
		return indicator.HashingAlgoHash
	case "string":
		return indicator.HashingAlgoString
	default:
		log.Warnf("Unknown hashing algorithm selected in %s: %q. Using default 'FNV64'.", v.EnvVar(), v.Setting())
		return indicator.HashingAlgoHash
	}
}

// NewTransitionBased creates a new instance of the transition-based update computer.
func NewTransitionBased() *TransitionBased {
	return &TransitionBased{
		hashingAlgo:                hashingAlgoFromEnv(env.NetworkFlowDeduperHashingAlgorithm),
		connectionsDeduper:         newStringSetPtr(),
		endpointsDeduper:           make(map[string]string),
		cachedUpdatesConn:          make([]*storage.NetworkFlow, 0),
		cachedUpdatesEp:            make([]*storage.NetworkEndpoint, 0),
		cachedUpdatesProc:          make([]*storage.ProcessListeningOnPortFromSensor, 0),
		closedConnTimestamps:       make(map[string]closedConnEntry),
		closedConnRememberDuration: env.NetworkFlowClosedConnRememberDuration.DurationSetting(),
		lastCleanup:                time.Now(),
	}
}

// ComputeUpdatedConns returns a list of network flow updates to be sent to Central.
func (c *TransitionBased) ComputeUpdatedConns(current map[indicator.NetworkConn]timestamp.MicroTS) []*storage.NetworkFlow {
	var updates []*storage.NetworkFlow
	ee := ConnectionEnrichedEntity
	if len(current) == 0 {
		// Received an empty map with current state. This may happen because:
		// - Some items were discarded during the enrichment process, so none made it through.
		// - This command was run on an empty map.
		// In this case, the current updates would be empty.
		// Return the cache as it may contain past updates collected during the offline mode.
		return c.cachedUpdatesConn
	}
	// Process each enriched connection individually, categorize the transition, and generate an update if needed.
	for conn, currTS := range current {
		key := conn.Key(c.hashingAlgo)

		// Check if this connection has been closed recently.
		prevTsFound, prevTS := c.lookupPrevTimestamp(key)
		// Based on the categorization, calculate the transition and determine if an update should be sent.
		update, transition := categorizeUpdate(prevTS, currTS, prevTsFound, key, c.connectionsDeduper, &c.connectionsDeduperMutex)
		updateMetrics(update, transition, ee)
		// Each transition may require updating the deduper.
		action := getConnectionDeduperAction(transition)
		switch action {
		case deduperActionAdd:
			c.connectionsDeduper.Add(key)
		case deduperActionRemove:
			c.connectionsDeduper.Remove(key)
		default: // noop
		}
		if update {
			c.storeClosedConnectionTimestamp(key, currTS, c.closedConnRememberDuration)
			updates = append(updates, conn.ToProto(currTS))
		}
	}
	// Store into cache in case sending to Central fails.
	c.cachedUpdatesConn = slices.Grow(c.cachedUpdatesConn, len(updates))
	c.cachedUpdatesConn = append(c.cachedUpdatesConn, updates...)
	// Return concatenated past and current updates.
	return c.cachedUpdatesConn
}

// categorizeUpdate determines whether an update to Central should be sent for a given enrichment update.
// The function is optimized to execute less expensive checks first and
// for readability (some conditions could be condensed but are kept separate for clarity).
// Note that enriched entities for which enrichment should be retried never reach this function.
func categorizeUpdate(
	prevTS, currTS timestamp.MicroTS, prevTsFound bool,
	connKey string,
	deduper *set.StringSet, mutex *sync.RWMutex) (bool, TransitionType) {

	// Variables for ease of reading
	isClosed := currTS != timestamp.InfiniteFuture
	wasClosed := prevTsFound && prevTS != timestamp.InfiniteFuture

	// CLOSED -> CLOSED
	if wasClosed && isClosed {
		// Update only if currTS is later than prevTS.
		if prevTS < currTS {
			return true, TransitionTypeClosed2Closed
		}
		return false, TransitionTypeClosed2Closed
	}
	// CLOSED -> OPEN
	if wasClosed {
		return true, TransitionTypeClosed2Open
	}
	if isClosed {
		// OPEN -> CLOSED (or NEW->CLOSED, it is actually the same)
		return true, TransitionTypeOpen2Closed
	}

	// OPEN -> OPEN - as last check due to costly search in the deduper
	seenPreviouslyOpen := concurrency.WithRLock1(mutex, func() bool {
		return deduper.Contains(connKey)
	})
	if seenPreviouslyOpen {
		return false, TransitionTypeOpen2Open
	}
	return true, TransitionTypeNew2Open

}

// getConnectionDeduperAction returns action to be executed on a deduper (or noop) for a given transition between states.
func getConnectionDeduperAction(tt TransitionType) deduperAction {
	switch tt {
	case TransitionTypeOpen2Closed:
		// When a previously open EE is being closed, we must remove it from the deduper.
		return deduperActionRemove
	case TransitionTypeNew2Open:
		// Add to deduper if open EE is seen for the first time.
		return deduperActionAdd
	case TransitionTypeClosed2Open:
		// Rarity. An EE was closed in the previous tick, but now is open.
		// We treat is as a new EE and thus add it to deduper.
		return deduperActionAdd
	default:
		// All other cases:
		// 1. Closed -> Closed - the first observation of closed EE would remove it from deduper.
		// 2. Open -> Open - the first observation of open EE would add it to deduper.
		return deduperActionNoop
	}
}

func updateMetrics(update bool, tt TransitionType, ee EnrichedEntity) {
	reason := skipReasonNone
	action := updateActionUpdate
	if !update {
		action = updateActionSkip
		// When no update should be sent, there are two major reasons for it.
		switch tt {
		case TransitionTypeClosed2Closed:
			reason = skipReasonTimestampOlder
		case TransitionTypeOpen2Open:
			reason = skipReasonAlreadySeen
		default:
			reason = skipReasonNone
		}
	}
	UpdateEvents.WithLabelValues(tt.String(), string(ee), action, reason).Inc()
}

// ComputeUpdatedEndpointsAndProcesses computes updates to Central for endpoints and their processes
func (c *TransitionBased) ComputeUpdatedEndpointsAndProcesses(
	enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp,
) ([]*storage.NetworkEndpoint, []*storage.ProcessListeningOnPortFromSensor) {
	if len(enrichedEndpointsProcesses) == 0 {
		// Received an empty map with current state. This may happen because:
		// - Some items were discarded during the enrichment process, so none made it through.
		// - This command was run on an empty map.
		// In this case, the current updates would be empty.
		// The `cachedUpdates` already contains past updates collected during offline mode, so no action needed.
		return c.cachedUpdatesEp, c.cachedUpdatesProc
	}
	plopEnabled := env.ProcessesListeningOnPort.BooleanSetting()

	var epUpdates []*storage.NetworkEndpoint
	var procUpdates []*storage.ProcessListeningOnPortFromSensor

	// Process currently enriched entities one by one, categorize the transition, and generate an update if applicable.
	for ep, p := range enrichedEndpointsProcesses {
		currTS := p.LastSeen
		epKey := ep.Key(c.hashingAlgo)
		// Check if this endpoint has a process.
		procKey := ""
		// If process was replaced (ep1->proc1 changed to ep1->proc2), the `procInd` would be the new process indicator.
		// There is currently no way to get the old processIndicator, so we don't inform Central about the old being closed.
		// If that is needed, this can be added in the future.
		if p.ProcessListening != nil {
			procKey = p.ProcessListening.Key(c.hashingAlgo)
		}
		// Based on the categorization, we calculate the transition and whether an update should be sent.
		sendEndpointUpdate, sendProcessUpdate, transition, dAction := categorizeEndpointUpdate(currTS, epKey, procKey, c.deduperHasEndpointAndProcess)
		updateMetrics(sendEndpointUpdate, transition, EndpointEnrichedEntity)
		updateMetrics(sendProcessUpdate, transition, ProcessEnrichedEntity)

		switch dAction {
		case deduperActionAdd, deduperActionUpdateProcess:
			concurrency.WithLock(&c.endpointsDeduperMutex, func() {
				c.endpointsDeduper[epKey] = procKey
			})
		case deduperActionRemove:
			concurrency.WithLock(&c.endpointsDeduperMutex, func() {
				delete(c.endpointsDeduper, epKey)
			})
		default: // noop
		}
		if sendEndpointUpdate {
			epUpdates = append(epUpdates, ep.ToProto(currTS))
		}
		if plopEnabled && sendProcessUpdate && p.ProcessListening != nil {
			procUpdates = append(procUpdates, p.ProcessListening.ToProto(currTS))
		}
	}

	// Store into cache in case sending to Central fails.
	c.cachedUpdatesEp = slices.Grow(c.cachedUpdatesEp, len(epUpdates))
	c.cachedUpdatesEp = append(c.cachedUpdatesEp, epUpdates...)
	if plopEnabled {
		c.cachedUpdatesProc = slices.Grow(c.cachedUpdatesProc, len(procUpdates))
		c.cachedUpdatesProc = append(c.cachedUpdatesProc, procUpdates...)
	}
	// Return concatenated past and current updates.
	return c.cachedUpdatesEp, c.cachedUpdatesProc
}

// categorizeEndpointUpdate determines whether an update to Central should be sent for a given enrichment update.
// The function is optimized to execute less expensive checks first and
// for readability (some conditions could be condensed but are kept separate for clarity).
// Note that enriched entities for which enrichment should be retried never reach this function.
//
// Unlike categorizeUpdate, this function does not consider the state from the previous tick. It only makes
// decisions based on data from the current tick (currently used for endpoints and processes).
// This approach simplifies the decision process and saves memory by not remembering recently closed entities.
//
// Returns a boolean indicating whether an update should be sent and a TransitionType describing
// the type of transition.
func categorizeEndpointUpdate(currTS timestamp.MicroTS, epKey, procKey string,
	deduperHas func(string, string) (bool, bool)) (updateEp, updateProc bool, tt TransitionType, da deduperAction) {
	// Variables for ease of reading
	isClosed := currTS != timestamp.InfiniteFuture

	// UNKNOWN -> CLOSED
	if isClosed {
		// We are unable to check whether this is closed->closed or open->closed.
		// The latter is assumed to be on the safe side and properly update the deduper.
		// Update to Central will always be sent, even if that was indeed a closed->closed transition.
		return true, true, TransitionTypeOpen2Closed, deduperActionRemove
	}
	// UNKNOWN -> OPEN - as last check due to costly search in the deduper
	knownEp, knownProc := deduperHas(epKey, procKey)
	if !knownEp {
		// This is a new, previously unseen endpoint, so the process must also be new. Send both updates.
		return true, true, TransitionTypeNew2Open, deduperActionAdd
	}
	if knownProc {
		// We have seen that endpoint and process together already. Skip updates.
		return false, false, TransitionTypeOpen2Open, deduperActionNoop
	}
	// We have seen that endpoint, but it had different process. We must update the process.
	return false, true, TransitionTypeReplaceProcess, deduperActionUpdateProcess
}

func (c *TransitionBased) deduperHasEndpointAndProcess(epKey, procKey string) (bool, bool) {
	return concurrency.WithRLock2(&c.endpointsDeduperMutex, func() (bool, bool) {
		pKey, ok := c.endpointsDeduper[epKey]
		if !ok {
			return false, false
		}
		return true, pKey == procKey
	})
}

func (c *TransitionBased) OnSuccessfulSendConnections(conns map[indicator.NetworkConn]timestamp.MicroTS) {
	if conns != nil {
		c.cachedUpdatesConn = make([]*storage.NetworkFlow, 0)
	}
}

// OnSuccessfulSendEndpoints updates the internal enrichedConnsLastSentState map with the currentState state.
// Providing nil will skip updates for respective map.
// Providing empty map will reset the state for given state.
func (c *TransitionBased) OnSuccessfulSendEndpoints(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
	if enrichedEndpointsProcesses != nil {
		c.cachedUpdatesEp = make([]*storage.NetworkEndpoint, 0)
	}
}

// OnSuccessfulSendProcesses contains actions that should be executed after successful sending of processesListening updates to Central.
func (c *TransitionBased) OnSuccessfulSendProcesses(enrichedEndpointsProcesses map[indicator.ContainerEndpoint]*indicator.ProcessListeningWithTimestamp) {
	if enrichedEndpointsProcesses != nil {
		c.cachedUpdatesProc = make([]*storage.ProcessListeningOnPortFromSensor, 0)
	}
}

func (c *TransitionBased) updateLastCleanup(now time.Time) {
	c.lastCleanupMutex.Lock()
	defer c.lastCleanupMutex.Unlock()
	c.lastCleanup = now
}

// ResetState clears the transition-based computer's firstTimeSeen tracking
func (c *TransitionBased) ResetState() {
	concurrency.WithLock(&c.connectionsDeduperMutex, func() {
		c.connectionsDeduper = newStringSetPtr()
	})
	concurrency.WithLock(&c.endpointsDeduperMutex, func() {
		c.endpointsDeduper = make(map[string]string)
	})

	c.updateLastCleanup(time.Now())

	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps = make(map[string]closedConnEntry)
	})
}

func (c *TransitionBased) RecordSizeMetrics(lenSize, byteSize *prometheus.GaugeVec) {
	valueConns := concurrency.WithRLock1(&c.connectionsDeduperMutex, func() int {
		return c.connectionsDeduper.Cardinality()
	})
	lenSize.WithLabelValues("deduper", string(ConnectionEnrichedEntity)).Set(float64(valueConns))

	valueEps := concurrency.WithRLock1(&c.endpointsDeduperMutex, func() int {
		return len(c.endpointsDeduper)
	})
	lenSize.WithLabelValues("deduper", string(EndpointEnrichedEntity)).Set(float64(valueEps))

	value := concurrency.WithRLock1(&c.closedConnMutex, func() int {
		return len(c.closedConnTimestamps)
	})
	lenSize.WithLabelValues("closedTimestamps", string(ConnectionEnrichedEntity)).Set(float64(value))

	// Calculate byte metrics
	byteSize.WithLabelValues("deduper", string(ConnectionEnrichedEntity)).Set(float64(c.calculateConnectionsDeduperByteSize()))
	byteSize.WithLabelValues("deduper", string(EndpointEnrichedEntity)).Set(float64(c.calculateEndpointsDeduperByteSize()))

	// Size of buffers that hold updates to Central while Sensor is offline
	lenSize.WithLabelValues("cachedUpdates", string(ConnectionEnrichedEntity)).Set(float64(len(c.cachedUpdatesConn)))
	lenSize.WithLabelValues("cachedUpdates", string(EndpointEnrichedEntity)).Set(float64(len(c.cachedUpdatesEp)))
	lenSize.WithLabelValues("cachedUpdates", string(ProcessEnrichedEntity)).Set(float64(len(c.cachedUpdatesProc)))
}

// lookupPrevTimestamp retrieves the previous close-timestamp for a connection.
// For open connections, returns found==false.
// For recently closed connections, returns the stored timestamp and found==true.
func (c *TransitionBased) lookupPrevTimestamp(connKey string) (found bool, prevTS timestamp.MicroTS) {
	// For closed connections, check if we have stored previous timestamp
	c.closedConnMutex.RLock()
	defer c.closedConnMutex.RUnlock()
	entry, exists := c.closedConnTimestamps[connKey]
	return exists, entry.prevTS
}

// storeClosedConnectionTimestamp stores the timestamp of a closed connection for future reference
func (c *TransitionBased) storeClosedConnectionTimestamp(
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

// calculateConnectionsDeduperByteSize calculates the memory usage of the connections deduper.
// The calculation includes: map reference (8 bytes) + string references (16 bytes per entry) + actual string content.
func (c *TransitionBased) calculateConnectionsDeduperByteSize() uintptr {
	baseSize := concurrency.WithRLock1(&c.connectionsDeduperMutex, func() uintptr {
		var totalStringBytes uintptr
		for _, s := range c.connectionsDeduper.AsSlice() {
			totalStringBytes += uintptr(len(s))
		}
		return uintptr(8) + // map reference
			uintptr(c.connectionsDeduper.Cardinality())*16 + // string references (16 bytes each)
			totalStringBytes // actual string content
	})

	// Conservative 2x multiplier for set.StringSet overhead (buckets, hash table structure, etc.)
	// The benchmarked overhead was 199/104 = 1.91x, but we use a slightly higher multiplier to be safe.
	return baseSize * 2
}

// calculateEndpointsDeduperByteSize calculates the memory usage of the endpoints deduper.
func (c *TransitionBased) calculateEndpointsDeduperByteSize() uintptr {
	baseSize := concurrency.WithRLock1(&c.endpointsDeduperMutex, func() uintptr {
		var totalStringBytes uintptr
		for k, v := range c.endpointsDeduper {
			totalStringBytes += uintptr(len(k) + len(v))
		}

		return uintptr(8) + // map reference
			uintptr(len(c.endpointsDeduper))*2*16 + // two string refs per entry (key + value), 16 bytes each
			totalStringBytes // actual string content
	})
	// Conservative 1.8x multiplier for Go map overhead (buckets, hash table structure, etc.)
	// The benchmarked overhead was 1.67x, but we use a slightly higher multiplier to be safe.
	return baseSize * 18 / 10
}

// PeriodicCleanup removes expired items from `closedConnTimestamps`.
func (c *TransitionBased) PeriodicCleanup(now time.Time, cleanupInterval time.Duration) {
	timer := prometheus.NewTimer(periodicCleanupDurationSeconds)
	defer timer.ObserveDuration()

	// Only run cleanup every minute to avoid excessive overhead
	concurrency.WithRLock(&c.lastCleanupMutex, func() {
		if now.Sub(c.lastCleanup) < cleanupInterval {
			return
		}
	})

	// Perform the cleanup
	concurrency.WithLock(&c.closedConnMutex, func() {
		for key, entry := range c.closedConnTimestamps {
			if timestamp.FromGoTime(now).After(entry.expiresAt) {
				delete(c.closedConnTimestamps, key)
			}
		}
	})
	c.updateLastCleanup(now)
}
