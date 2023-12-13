package clusterentities

func newEntityStatus(numTicks uint16) *entityStatus {
	return &entityStatus{
		ticksLeft:    numTicks,
		isHistorical: false,
	}
}

type entityStatus struct {
	ticksLeft    uint16
	isHistorical bool
}

// markHistorical is called when entity would be deleted.
// Istead, we mark it as historcal and keep it as long as ticksLeft
func (es *entityStatus) markHistorical(ticksLeft uint16) {
	if !es.isHistorical {
		es.ticksLeft = ticksLeft
	}
	es.isHistorical = true
}

// recordTick decreases value of ticksLeft until it reaches 0
func (es *entityStatus) recordTick() {
	if !es.isHistorical {
		return
	}
	if es.ticksLeft > 0 {
		es.ticksLeft--
	}
}

// IsExpired returns true if historical entry waited for `ticksLeft` ticks
func (es *entityStatus) IsExpired() bool {
	return es.isHistorical && es.ticksLeft == 0
}
