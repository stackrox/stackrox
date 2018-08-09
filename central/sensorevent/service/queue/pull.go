package queue

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// Pull attempts to get the next item from the queue.
// When no items are available, both the event and error will be nil.
// If anything goes wrong, and error will be returned.
func (p *persistedEventQueue) Pull() (*v1.SensorEvent, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for len(p.seqIDQueue) == 0 {
		return nil, nil
	}

	// Get the next id and remove it from the id queue.
	id := p.seqIDQueue[0]
	p.seqIDQueue = p.seqIDQueue[1:]

	// Attempt to load the event data from persistence.
	event, exists, err := p.eventStorage.GetSensorEvent(id)
	if err != nil {
		return nil, fmt.Errorf("unable to pull next event from db: %s", err)
	}
	if !exists {
		return nil, fmt.Errorf("next event does not exist in db: %d", id)
	}

	// We need to remove its id mapping when we find it so we don't have it dangling around.
	delete(p.depIDToSeqID, event.GetId())

	// Try to remove the event from storage since we are returning it, but return the event either way.
	if err := p.eventStorage.RemoveSensorEvent(id); err != nil {
		return event, fmt.Errorf("cannot remove next event from the db: %s", err)
	}
	return event, nil
}
