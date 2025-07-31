package manager

import (
	"fmt"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
)

// UpdateComputerType represents the type of update computer to use
type UpdateComputerType string

const (
	// LegacyUpdateComputerType uses the original LastSentState-based logic
	LegacyUpdateComputerType UpdateComputerType = "legacy"
	// CategorizedUpdateComputerType uses the new categorized update logic
	CategorizedUpdateComputerType UpdateComputerType = "categorized"
)

// UpdateComputer defines the interface for computing network flow updates to send to Central
// Each implementation manages its own state and computation strategy
type UpdateComputer interface {
	// Compute updates based on current state and implementation-specific tracking
	ComputeUpdatedConns(current map[networkConnIndicator]timestamp.MicroTS) []*storage.NetworkFlow
	ComputeUpdatedEndpoints(current map[containerEndpointIndicator]timestamp.MicroTS) []*storage.NetworkEndpoint
	ComputeUpdatedProcesses(current map[processListeningIndicator]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor

	// State management - each implementation handles its own state updates
	UpdateState(currentConns map[networkConnIndicator]timestamp.MicroTS, currentEndpoints map[containerEndpointIndicator]timestamp.MicroTS, currentProcesses map[processListeningIndicator]timestamp.MicroTS)

	// Reset all internal state (used when clearing historical data)
	ResetState()

	// Get metrics about internal state size for monitoring
	GetStateMetrics() (connsSize, endpointsSize, processesSize int)
}

// LegacyUpdateComputer implements the original update computation logic using LastSentState maps
// It owns and manages the LastSentState maps that were previously in the manager
type LegacyUpdateComputer struct {
	// State tracking maps - these were previously in networkFlowManager
	enrichedConnsLastSentState     map[networkConnIndicator]timestamp.MicroTS
	enrichedEndpointsLastSentState map[containerEndpointIndicator]timestamp.MicroTS
	enrichedProcessesLastSentState map[processListeningIndicator]timestamp.MicroTS

	// Mutex to protect the LastSentState maps
	lastSentStateMutex sync.RWMutex
}

// NewLegacyUpdateComputer creates a new instance of the legacy update computer
func NewLegacyUpdateComputer() UpdateComputer {
	return &LegacyUpdateComputer{
		enrichedConnsLastSentState:     make(map[networkConnIndicator]timestamp.MicroTS),
		enrichedEndpointsLastSentState: make(map[containerEndpointIndicator]timestamp.MicroTS),
		enrichedProcessesLastSentState: make(map[processListeningIndicator]timestamp.MicroTS),
	}
}

func (l *LegacyUpdateComputer) ComputeUpdatedConns(current map[networkConnIndicator]timestamp.MicroTS) []*storage.NetworkFlow {
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()
	var updates []*storage.NetworkFlow

	for conn, currTS := range current {
		prevTS, seenPreviously := l.enrichedConnsLastSentState[conn]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, conn.toProto(currTS))
		}
	}

	for conn, prevTS := range l.enrichedConnsLastSentState {
		if _, ok := current[conn]; !ok {
			updates = append(updates, conn.toProto(prevTS))
		}
	}

	return updates
}

func (l *LegacyUpdateComputer) ComputeUpdatedEndpoints(current map[containerEndpointIndicator]timestamp.MicroTS) []*storage.NetworkEndpoint {
	l.lastSentStateMutex.RLock()
	defer l.lastSentStateMutex.RUnlock()
	var updates []*storage.NetworkEndpoint

	for ep, currTS := range current {
		prevTS, seenPreviously := l.enrichedEndpointsLastSentState[ep]
		if isUpdated(prevTS, currTS, seenPreviously) {
			updates = append(updates, ep.toProto(currTS))
		}
	}

	for ep, prevTS := range l.enrichedEndpointsLastSentState {
		if _, ok := current[ep]; !ok {
			updates = append(updates, ep.toProto(prevTS))
		}
	}

	return updates
}

func (l *LegacyUpdateComputer) ComputeUpdatedProcesses(current map[processListeningIndicator]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
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
			updates = append(updates, pl.toProto(currTS))
		}
	}

	for ep, prevTS := range l.enrichedProcessesLastSentState {
		if _, ok := current[ep]; !ok {
			// This condition means the deployment was removed before we got the
			// close timestamp for the endpoint. Use the current timestamp instead.
			if prevTS == timestamp.InfiniteFuture {
				prevTS = timestamp.Now()
			}
			updates = append(updates, ep.toProto(prevTS))
		}
	}

	return updates
}

// UpdateState updates the internal LastSentState maps with the current state
func (l *LegacyUpdateComputer) UpdateState(currentConns map[networkConnIndicator]timestamp.MicroTS, currentEndpoints map[containerEndpointIndicator]timestamp.MicroTS, currentProcesses map[processListeningIndicator]timestamp.MicroTS) {
	l.lastSentStateMutex.Lock()
	defer l.lastSentStateMutex.Unlock()

	// Update connections state
	l.enrichedConnsLastSentState = make(map[networkConnIndicator]timestamp.MicroTS, len(currentConns))
	for conn, ts := range currentConns {
		l.enrichedConnsLastSentState[conn] = ts
	}

	// Update endpoints state
	l.enrichedEndpointsLastSentState = make(map[containerEndpointIndicator]timestamp.MicroTS, len(currentEndpoints))
	for ep, ts := range currentEndpoints {
		l.enrichedEndpointsLastSentState[ep] = ts
	}

	// Update processes state
	l.enrichedProcessesLastSentState = make(map[processListeningIndicator]timestamp.MicroTS, len(currentProcesses))
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

// CategorizedUpdateComputer implements the new categorized update computation logic
// It owns and manages the firstTimeSeen tracking that was previously in the manager
type CategorizedUpdateComputer struct {
	manager *networkFlowManager // Reference to access core manager functionality

	// State tracking for conditional updates - moved from networkFlowManager
	conditionalUpdatesMutex sync.RWMutex
	firstTimeSeenConns      set.StringSet
	firstTimeSeenEndpoints  set.StringSet
	firstTimeSeenProcesses  set.StringSet
}

// NewCategorizedUpdateComputer creates a new instance of the categorized update computer
func NewCategorizedUpdateComputer(manager *networkFlowManager) UpdateComputer {
	return &CategorizedUpdateComputer{
		manager:                manager,
		firstTimeSeenConns:     set.NewStringSet(),
		firstTimeSeenEndpoints: set.NewStringSet(),
		firstTimeSeenProcesses: set.NewStringSet(),
	}
}

func (c *CategorizedUpdateComputer) ComputeUpdatedConns(current map[networkConnIndicator]timestamp.MicroTS) []*storage.NetworkFlow {
	// Use the categorized computer's own categorization logic and firstTimeSeen tracking
	var updates []*storage.NetworkFlow
	var closedConnKeys []string

	// Process current connections using our own categorization
	for conn, currTS := range current {
		// For categorized implementation, we don't use previous state - categorization is based on internal tracking
		category := c.categorizeConnectionUpdate(conn, currTS, 0, false)

		switch category {
		case RequiredUpdate, ConditionalUpdate:
			updates = append(updates, conn.toProto(currTS))
			// If this is a closed connection, track it for cleanup
			if currTS != timestamp.InfiniteFuture {
				closedConnKeys = append(closedConnKeys, c.connectionKey(conn))
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

func (c *CategorizedUpdateComputer) ComputeUpdatedEndpoints(current map[containerEndpointIndicator]timestamp.MicroTS) []*storage.NetworkEndpoint {
	var updates []*storage.NetworkEndpoint
	var closedEndpointKeys []string

	for ep, currTS := range current {
		category := c.categorizeEndpointUpdate(ep, currTS, 0, false)

		switch category {
		case RequiredUpdate, ConditionalUpdate:
			updates = append(updates, ep.toProto(currTS))
			if currTS != timestamp.InfiniteFuture {
				closedEndpointKeys = append(closedEndpointKeys, c.endpointKey(ep))
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

func (c *CategorizedUpdateComputer) ComputeUpdatedProcesses(current map[processListeningIndicator]timestamp.MicroTS) []*storage.ProcessListeningOnPortFromSensor {
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
			updates = append(updates, proc.toProto(currTS))
			if currTS != timestamp.InfiniteFuture {
				closedProcessKeys = append(closedProcessKeys, c.processKey(proc))
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
// The categorization logic in the manager handles its own state updates
func (c *CategorizedUpdateComputer) UpdateState(currentConns map[networkConnIndicator]timestamp.MicroTS, currentEndpoints map[containerEndpointIndicator]timestamp.MicroTS, currentProcesses map[processListeningIndicator]timestamp.MicroTS) {
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
// Moved from networkFlowManager to CategorizedUpdateComputer to encapsulate state access
func (c *CategorizedUpdateComputer) categorizeConnectionUpdate(conn networkConnIndicator, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
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
	connKey := c.connectionKey(conn)
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
func (c *CategorizedUpdateComputer) categorizeEndpointUpdate(ep containerEndpointIndicator, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
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
	epKey := c.endpointKey(ep)
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
func (c *CategorizedUpdateComputer) categorizeProcessUpdate(proc processListeningIndicator, currTS timestamp.MicroTS, prevTS timestamp.MicroTS, seenPreviously bool) UpdateCategory {
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
	procKey := c.processKey(proc)
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

// connectionKey generates a unique string key for connection tracking
// Moved from networkFlowManager to CategorizedUpdateComputer
func (c *CategorizedUpdateComputer) connectionKey(conn networkConnIndicator) string {
	return fmt.Sprintf("%d:%s|%d:%s|%d|%d",
		int(conn.srcEntity.Type), conn.srcEntity.ID,
		int(conn.dstEntity.Type), conn.dstEntity.ID,
		conn.dstPort, int(conn.protocol))
}

// endpointKey generates a unique string key for endpoint tracking
func (c *CategorizedUpdateComputer) endpointKey(ep containerEndpointIndicator) string {
	return fmt.Sprintf("%d:%s|%d|%d",
		int(ep.entity.Type), ep.entity.ID,
		ep.port, int(ep.protocol))
}

// processKey generates a unique string key for process tracking
func (c *CategorizedUpdateComputer) processKey(proc processListeningIndicator) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%d|%d|%s|%s",
		proc.key.podID, proc.key.containerName, proc.key.deploymentID,
		proc.key.process.processName, proc.key.process.processExec,
		proc.port, int(proc.protocol),
		proc.podUID, proc.namespace)
}

// cleanupConditionalUpdateTracking removes tracking entries for closed connections/endpoints
// Moved from networkFlowManager to CategorizedUpdateComputer to encapsulate state management
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

// Option functions for configuring the update computer

// WithLegacyUpdateComputer configures the manager to use the legacy update computation logic
func WithLegacyUpdateComputer() Option {
	return func(mgr *networkFlowManager) {
		mgr.updateComputer = NewLegacyUpdateComputer()
	}
}

// WithCategorizedUpdateComputer configures the manager to use the new categorized update computation logic
func WithCategorizedUpdateComputer() Option {
	return func(mgr *networkFlowManager) {
		mgr.updateComputer = NewCategorizedUpdateComputer(mgr)
	}
}

// WithUpdateComputerType configures the manager to use the specified update computer type
func WithUpdateComputerType(updateType UpdateComputerType) Option {
	return func(mgr *networkFlowManager) {
		switch updateType {
		case LegacyUpdateComputerType:
			mgr.updateComputer = NewLegacyUpdateComputer()
		case CategorizedUpdateComputerType:
			mgr.updateComputer = NewCategorizedUpdateComputer(mgr)
		default:
			log.Warnf("Unknown update computer type %q, defaulting to categorized", updateType)
			mgr.updateComputer = NewCategorizedUpdateComputer(mgr)
		}
	}
}
