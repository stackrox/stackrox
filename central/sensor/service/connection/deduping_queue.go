package connection

import (
	"container/list"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/sync"
)

type dedupingQueue struct {
	mutex             sync.Mutex
	notEmptySig       concurrency.Signal
	queue             *list.List
	resourceIDToEvent map[string]*list.Element
	typ               string
}

func newDedupingQueue(typ string) *dedupingQueue {
	return &dedupingQueue{
		notEmptySig:       concurrency.NewSignal(),
		queue:             list.New(),
		resourceIDToEvent: make(map[string]*list.Element),
		typ:               typ,
	}
}

func (q *dedupingQueue) pullBlocking(abort concurrency.Waitable) *central.MsgFromSensor {
	var msg *central.MsgFromSensor
	for msg == nil {
		select {
		case <-abort.Done():
			return nil
		case <-q.notEmptySig.Done():
			msg = q.pull()
		}
	}
	return msg
}

func (q *dedupingQueue) pull() *central.MsgFromSensor {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.queue.Len() == 0 {
		return nil
	}

	msg := q.queue.Remove(q.queue.Front()).(*central.MsgFromSensor)
	metrics.IncrementSensorEventQueueCounter(ops.Remove, q.typ)

	if msg.GetDedupeKey() != "" {
		delete(q.resourceIDToEvent, msg.GetDedupeKey())
	}

	if q.queue.Len() == 0 {
		q.notEmptySig.Reset()
	}
	return msg
}

// push attempts to add an item to the queue, and returns an error if it is unable.
func (q *dedupingQueue) push(msg *central.MsgFromSensor) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	defer q.notEmptySig.Signal()
	q.pushNoLock(msg)
}

func (q *dedupingQueue) pushNoLock(msg *central.MsgFromSensor) {
	if msg.GetDedupeKey() == "" {
		metrics.IncrementSensorEventQueueCounter(ops.Add, q.typ)
		q.queue.PushBack(msg)
		return
	}
	var msgInserted *list.Element
	if evt, ok := q.resourceIDToEvent[msg.GetDedupeKey()]; ok {
		metrics.IncrementSensorEventQueueCounter(ops.Dedupe, q.typ)
		msgInserted = q.queue.InsertBefore(msg, evt)
		q.queue.Remove(evt)
	} else {
		metrics.IncrementSensorEventQueueCounter(ops.Add, q.typ)
		msgInserted = q.queue.PushBack(msg)
	}
	q.resourceIDToEvent[msg.GetDedupeKey()] = msgInserted
}
