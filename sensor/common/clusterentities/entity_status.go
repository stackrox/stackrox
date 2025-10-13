package clusterentities

func newHistoricalEntity(numTicks uint16) *entityStatus {
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
