package queue

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// Push attempts to add an item to the queue, and returns an error if it is unable.
func (p *persistedEventQueue) Push(event *v1.SensorEvent) error {
	// We can ignore any unrecognized or unhandled actions.
	if handled, known := handledActions[event.GetAction()]; !handled || !known {
		return fmt.Errorf("unable to handle action %d", event.GetAction())
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Add the item to the queue, or if it is already in the queue, run deduplication on it.
	seqID, exists := p.depIDToSeqID[event.GetId()]
	if exists {
		return p.pushExistingLockFree(event, seqID)
	}
	return p.pushNewLockFree(event)
}

// Push a new event into the queue.
func (p *persistedEventQueue) pushNewLockFree(event *v1.SensorEvent) error {
	id, err := p.eventStorage.AddSensorEvent(event)
	if err != nil {
		return fmt.Errorf("unable to add event to db: %s", err)
	}

	p.depIDToSeqID[event.GetId()] = id
	p.seqIDQueue = append(p.seqIDQueue, id)
	return nil
}

// Update an existing event in the queue
func (p *persistedEventQueue) pushExistingLockFree(event *v1.SensorEvent, seqID uint64) error {
	// Get the current deployment stored in the queue.
	currentEvent, exists, err := p.eventStorage.GetSensorEvent(seqID)
	if err != nil {
		return fmt.Errorf("unable to fetch stored event: %s", err)
	}
	if !exists {
		delete(p.depIDToSeqID, event.GetId())
		return fmt.Errorf("sequence stored but event missing: %s", err)
	}

	// Try to deduplicate the event, either by updating the stored data or by removing it from the queue.
	updateToApply, removeInstead, err := p.dedupeEvents(currentEvent, event)
	if err != nil {
		return fmt.Errorf("unable to dedupe event: %s", err)
	}
	if updateToApply != nil {
		return p.eventStorage.UpdateSensorEvent(seqID, updateToApply)
	}
	if removeInstead {
		return p.remove(seqID)
	}
	return nil
}

// dedupeEvents takes in two events, one already pending, and one we want to add to the set of pending, and
// returns 3 values indicating if the current queue data needs to be update, removed, or if the sequence
// cannot be handled (error condition).
func (p *persistedEventQueue) dedupeEvents(firstEvent *v1.SensorEvent, secondEvent *v1.SensorEvent) (*v1.SensorEvent, bool, error) {
	// Sequences that require the old action on new data.
	if firstEvent.GetAction() == v1.ResourceAction_CREATE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_PREEXISTING_RESOURCE {
		secondEvent.Action = v1.ResourceAction_CREATE_RESOURCE
		return secondEvent, false, nil
	}
	if firstEvent.GetAction() == v1.ResourceAction_CREATE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_UPDATE_RESOURCE {
		secondEvent.Action = v1.ResourceAction_CREATE_RESOURCE
		return secondEvent, false, nil
	}
	if firstEvent.GetAction() == v1.ResourceAction_PREEXISTING_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_UPDATE_RESOURCE {
		return secondEvent, false, nil
	}

	// Sequences that replace the old action and data.
	if firstEvent.GetAction() == v1.ResourceAction_UPDATE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_UPDATE_RESOURCE {
		return secondEvent, false, nil
	}
	if firstEvent.GetAction() == v1.ResourceAction_UPDATE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		return secondEvent, false, nil
	}
	if firstEvent.GetAction() == v1.ResourceAction_REMOVE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		return secondEvent, false, nil
	}

	// Sequences that remove an item from the queue.
	if firstEvent.GetAction() == v1.ResourceAction_CREATE_RESOURCE && secondEvent.GetAction() == v1.ResourceAction_REMOVE_RESOURCE {
		return nil, true, nil
	}

	// Unrecognized duplication sequence, just return an error.
	return nil, false, fmt.Errorf("unhandled duplicate deployment sequence: %s %s", firstEvent.GetAction(), secondEvent.GetAction())
}
