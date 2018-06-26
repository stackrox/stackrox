package queue

import "fmt"

// Removes an event from the queue by deployment id if one exists.
func (p *persistedEventQueue) remove(seqID uint64) error {
	// If we can't load the event, or it doesn't exist, try the next id if one exists.
	event, exists, err := p.eventStorage.GetDeploymentEvent(seqID)
	if err != nil {
		return fmt.Errorf("unable to pull next event from db: %s", err)
	}
	if !exists {
		return fmt.Errorf("next event does not exist in db: %d", seqID)
	}

	delete(p.depIDToSeqID, event.GetDeployment().GetId())
	p.removeFromSeq(seqID)

	// Try to remove the event from storage since we are returning it.
	if err := p.eventStorage.RemoveDeploymentEvent(seqID); err != nil {
		return fmt.Errorf("cannot remove next event from the db: %s", err)
	}
	return nil
}

// Removes an id from the queue
func (p *persistedEventQueue) removeFromSeq(idToRemove uint64) {
	for index, id := range p.seqIDQueue {
		if id == idToRemove {
			if index == 0 {
				p.seqIDQueue = p.seqIDQueue[index+1:]
			} else if index == (len(p.seqIDQueue) - 1) {
				p.seqIDQueue = p.seqIDQueue[:index]
			} else {
				p.seqIDQueue = append(p.seqIDQueue[:index], p.seqIDQueue[index+1:]...)
			}
		}
	}
}
