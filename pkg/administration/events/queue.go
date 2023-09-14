package events

import (
	"container/list"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
)

const (
	maxQueueSize = 100000
)

type administrationEventsQueue struct {
	mutex       sync.Mutex
	queue       *list.List
	notEmptySig concurrency.Signal
}

func newQueue() *administrationEventsQueue {
	return &administrationEventsQueue{
		notEmptySig: concurrency.NewSignal(),
		queue:       list.New(),
	}
}

func (q *administrationEventsQueue) pullBlocking(waitable concurrency.Waitable) *AdministrationEvent {
	var event *AdministrationEvent
	for event == nil {
		select {
		case <-waitable.Done():
			return nil
		case <-q.notEmptySig.Done():
			event = q.pull()
		}
	}
	return event
}

func (q *administrationEventsQueue) pull() *AdministrationEvent {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() == 0 {
		return nil
	}

	event := q.queue.Remove(q.queue.Front()).(*AdministrationEvent)

	if q.queue.Len() == 0 {
		q.notEmptySig.Reset()
	}
	return event
}

func (q *administrationEventsQueue) push(event *AdministrationEvent) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() >= maxQueueSize {
		return
	}

	defer q.notEmptySig.Signal()
	q.queue.PushBack(event)
}
