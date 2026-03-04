package clusterentities

// newHistoricalEntity creates a new historical entity with the specified number of ticks.
// For endpointsStore, this returns a uint16 directly to eliminate heap allocation overhead.
// Other stores still use the entityStatus struct via newHistoricalEntityPtr.
func newHistoricalEntity(numTicks uint16) uint16 {
	return numTicks
}

// newHistoricalEntityPtr creates a historical entity as a pointer (legacy approach).
// Used by podIPsStore and containerIDsStore which haven't been optimized yet.
func newHistoricalEntityPtr(numTicks uint16) *entityStatus {
	return &entityStatus{
		ticksLeft: numTicks,
	}
}

type entityStatus struct {
	ticksLeft uint16
}

// recordTick decreases the value of ticksLeft until it reaches 0
func (es *entityStatus) recordTick() {
	if es.ticksLeft > 0 {
		es.ticksLeft--
	}
}

// IsExpired returns true if historical entry waited for `ticksLeft` ticks
func (es *entityStatus) IsExpired() bool {
	return es.ticksLeft == 0
}
