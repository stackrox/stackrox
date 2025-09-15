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

// allEnrichedEntities is a list of all enriched entities. It is used to gather metrics for all enriched entities types.
var allEnrichedEntities = []EnrichedEntity{
	ConnectionEnrichedEntity,
	EndpointEnrichedEntity,
	ProcessEnrichedEntity}

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
	deduperMutex    sync.RWMutex
	deduper         map[EnrichedEntity]*set.StringSet
	deduperEstBytes map[EnrichedEntity]uintptr

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
		hashingAlgo: hashingAlgoFromEnv(env.NetworkFlowDeduperHashingAlgorithm),
		deduper: map[EnrichedEntity]*set.StringSet{
			ConnectionEnrichedEntity: newStringSetPtr(),
			EndpointEnrichedEntity:   newStringSetPtr(),
			ProcessEnrichedEntity:    newStringSetPtr(),
		},
		deduperEstBytes: map[EnrichedEntity]uintptr{
			ConnectionEnrichedEntity: 0,
			EndpointEnrichedEntity:   0,
			ProcessEnrichedEntity:    0,
		},
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
		update, transition := categorizeUpdate(prevTS, currTS, prevTsFound, key, ee, c.deduper, &c.deduperMutex)
		updateMetrics(update, transition, ee)
		// Each transition may require updating the deduper.
		action := getDeduperAction(transition)
		switch action {
		case deduperActionAdd:
			c.deduper[ee].Add(key)
		case deduperActionRemove:
			c.deduper[ee].Remove(key)
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
	connKey string, ee EnrichedEntity,
	deduper map[EnrichedEntity]*set.StringSet, mutex *sync.RWMutex) (bool, TransitionType) {

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
		return deduper[ee].Contains(connKey)
	})
	if seenPreviouslyOpen {
		return false, TransitionTypeOpen2Open
	}
	return true, TransitionTypeNew2Open

}

// getDeduperAction returns action to be executed on a deduper (or noop) for a given transition between states.
func getDeduperAction(tt TransitionType) deduperAction {
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

// categorizeUpdateNoPast determines whether an update to Central should be sent for a given enrichment update.
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
func categorizeUpdateNoPast(
	currTS timestamp.MicroTS,
	key string, deduper *set.StringSet, mutex *sync.RWMutex) (bool, TransitionType) {

	// Variables for ease of reading
	isClosed := currTS != timestamp.InfiniteFuture

	// UNKNOWN -> CLOSED
	if isClosed {
		// We are unable to check whether this is closed->closed or open->closed.
		// The latter is assumed to be on the safe side and properly update the deduper.
		// Update to Central will always be sent, even if that was indeed a closed->closed transition.
		return true, TransitionTypeOpen2Closed
	}
	// UNKNOWN -> OPEN - as last check due to costly search in the deduper
	seenPreviouslyOpen := concurrency.WithRLock1(mutex, func() bool {
		return deduper.Contains(key)
	})
	if seenPreviouslyOpen {
		return false, TransitionTypeOpen2Open
	}
	return true, TransitionTypeNew2Open
}

// ComputeUpdatedEndpoints computes endpoint updates to send to Central in the current tick.
// This method doesn't rely on the state of closed endpoints from the past; each closed endpoint generates an update.
func (c *TransitionBased) ComputeUpdatedEndpoints(current map[indicator.ContainerEndpoint]timestamp.MicroTS) []*storage.NetworkEndpoint {
	updates := computeUpdatedEntitiesNoPast(
		current,
		EndpointEnrichedEntity,
		c.deduper[EndpointEnrichedEntity],
		&c.deduperMutex,
		c.hashingAlgo,
		func(ep indicator.ContainerEndpoint, algo indicator.HashingAlgo) string {
			return ep.Key(algo)
		},
		func(ep indicator.ContainerEndpoint, ts timestamp.MicroTS) *storage.NetworkEndpoint {
			return ep.ToProto(ts)
		},
	)
	// Store into cache in case sending to Central fails.
	c.cachedUpdatesEp = slices.Grow(c.cachedUpdatesEp, len(updates))
	c.cachedUpdatesEp = append(c.cachedUpdatesEp, updates...)
	// Return concatenated past and current updates.
	return c.cachedUpdatesEp
}

// ComputeUpdatedProcesses computes process updates to send to Central in the current tick.
// This method doesn't rely on the state of closed processes from the past; each closed process generates an update.
func (c *TransitionBased) ComputeUpdatedProcesses(current map[indicator.ProcessListening]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
	if !env.ProcessesListeningOnPort.BooleanSetting() {
		if len(current) > 0 {
			logging.GetRateLimitedLogger().WarnL(loggingRateLimiter,
				"Received process(es) while ProcessesListeningOnPorts feature is disabled. This may indicate a misconfiguration.")
		}
		return []*storage.ProcessListeningOnPortFromSensor{}
	}

	updates := computeUpdatedEntitiesNoPast(
		current,
		ProcessEnrichedEntity,
		c.deduper[ProcessEnrichedEntity],
		&c.deduperMutex,
		c.hashingAlgo,
		func(proc indicator.ProcessListening, algo indicator.HashingAlgo) string {
			return proc.Key(algo)
		},
		func(proc indicator.ProcessListening, ts timestamp.MicroTS) *storage.ProcessListeningOnPortFromSensor {
			return proc.ToProto(ts)
		},
	)
	// Store into cache in case sending to Central fails.
	c.cachedUpdatesProc = slices.Grow(c.cachedUpdatesProc, len(updates))
	c.cachedUpdatesProc = append(c.cachedUpdatesProc, updates...)
	// Return concatenated past and current updates.
	return c.cachedUpdatesProc
}

// computeUpdatedEntitiesNoPast is a generic function that computes updates for any entity type.
// It eliminates code duplication between endpoints and processes by abstracting the common computation logic.
// The function operates directly on the cachedUpdates slice through a pointer for efficiency and clarity.
func computeUpdatedEntitiesNoPast[indicatorT comparable, updateT any](
	currentUpdates map[indicatorT]timestamp.MicroTS,
	ee EnrichedEntity,
	deduper *set.StringSet,
	deduperMutex *sync.RWMutex,
	hashingAlgo indicator.HashingAlgo,
	keyFunc func(indicatorT, indicator.HashingAlgo) string,
	toProto func(indicatorT, timestamp.MicroTS) updateT,
) []updateT {
	var updates []updateT
	if len(currentUpdates) == 0 {
		// Received an empty map with current state. This may happen because:
		// - Some items were discarded during the enrichment process, so none made it through.
		// - This command was run on an empty map.
		// In this case, the current updates would be empty.
		// The `cachedUpdates` already contains past updates collected during offline mode, so no action needed.
		return updates
	}

	// Process currently enriched entities one by one, categorize the transition, and generate an update if applicable.
	for entity, currTS := range currentUpdates {
		key := keyFunc(entity, hashingAlgo)
		// Based on the categorization, we calculate the transition and whether an update should be sent.
		update, transition := categorizeUpdateNoPast(currTS, key, deduper, deduperMutex)
		updateMetrics(update, transition, ee)
		// Each transition may require updating the deduper.
		// We cannot update the deduper right away, because Central may be offline, so we must store the operations
		// to execute on the deduper and execute them only when sending to Central is successful.
		action := getDeduperAction(transition)
		switch action {
		case deduperActionAdd:
			deduper.Add(key)
		case deduperActionRemove:
			deduper.Remove(key)
		default: // noop
		}
		if update {
			updates = append(updates, toProto(entity, currTS))
		}
	}
	return updates
}

// OnSuccessfulSend clears the cached updates to Central.
func (c *TransitionBased) OnSuccessfulSend(conns map[indicator.NetworkConn]timestamp.MicroTS,
	eps map[indicator.ContainerEndpoint]timestamp.MicroTS,
	procs map[indicator.ProcessListening]timestamp.MicroTS,
) {
	if conns != nil {
		c.cachedUpdatesConn = make([]*storage.NetworkFlow, 0)
	}
	if eps != nil {
		c.cachedUpdatesEp = make([]*storage.NetworkEndpoint, 0)
	}
	if procs != nil {
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
	concurrency.WithLock(&c.deduperMutex, func() {
		c.deduper = map[EnrichedEntity]*set.StringSet{
			ConnectionEnrichedEntity: newStringSetPtr(),
			EndpointEnrichedEntity:   newStringSetPtr(),
			ProcessEnrichedEntity:    newStringSetPtr(),
		}
		c.deduperEstBytes = map[EnrichedEntity]uintptr{
			ConnectionEnrichedEntity: 0,
			EndpointEnrichedEntity:   0,
			ProcessEnrichedEntity:    0,
		}
	})

	c.updateLastCleanup(time.Now())

	concurrency.WithLock(&c.closedConnMutex, func() {
		c.closedConnTimestamps = make(map[string]closedConnEntry)
	})
}

func (c *TransitionBased) RecordSizeMetrics(lenSize, byteSize *prometheus.GaugeVec) {
	for _, entity := range allEnrichedEntities {
		value := concurrency.WithRLock1(&c.deduperMutex, func() int {
			return c.deduper[entity].Cardinality()
		})
		lenSize.WithLabelValues("deduper", string(entity)).Set(float64(value))
	}
	value := concurrency.WithRLock1(&c.closedConnMutex, func() int {
		return len(c.closedConnTimestamps)
	})
	lenSize.WithLabelValues("closedTimestamps", string(ConnectionEnrichedEntity)).Set(float64(value))

	// Calculate byte metrics
	for _, entity := range allEnrichedEntities {
		baseSize := concurrency.WithRLock1(&c.deduperMutex, func() uintptr {
			var totalStringBytes uintptr
			for _, s := range c.deduper[entity].AsSlice() {
				totalStringBytes += uintptr(len(s))
			}
			return 8 + uintptr(c.deduper[entity].Cardinality())*16 + totalStringBytes
		})
		c.deduperEstBytes[entity] = baseSize * 2 // *2 comes from the overhead for map
		byteSize.WithLabelValues("deduper", string(entity)).Set(float64(c.deduperEstBytes[entity]))
	}

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

func (c *TransitionBased) HandlePurgedConnectionIndicator(conn *indicator.NetworkConn) {
	c.handleDeleted(conn, ConnectionEnrichedEntity)
}

func (c *TransitionBased) HandlePurgedEndpointIndicator(ep *indicator.ContainerEndpoint) {
	c.handleDeleted(ep, EndpointEnrichedEntity)
}

func (c *TransitionBased) HandlePurgedProcessIndicator(proc *indicator.ProcessListening) {
	c.handleDeleted(proc, ProcessEnrichedEntity)
}

type keyable interface {
	Key(indicator.HashingAlgo) string
}

func (c *TransitionBased) handleDeleted(item keyable, ee EnrichedEntity) {
	key := item.Key(c.hashingAlgo)
	concurrency.WithLock(&c.deduperMutex, func() {
		c.deduper[ee].Remove(key)
	})

}
