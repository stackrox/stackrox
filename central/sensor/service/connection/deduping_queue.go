package connection

import (
	"container/list"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
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
}

func newDedupingQueue() *dedupingQueue {
	return &dedupingQueue{
		notEmptySig:       concurrency.NewSignal(),
		queue:             list.New(),
		resourceIDToEvent: make(map[string]*list.Element),
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
	metrics.IncrementSensorEventQueueCounter(ops.Remove, common.GetMessageType(msg))

	if msg.GetDedupeKey() != "" {
		delete(q.resourceIDToEvent, msg.GetDedupeKey())
	}

	if q.queue.Len() == 0 {
		q.notEmptySig.Reset()
	}
	return msg
}

// Push attempts to add an item to the queue, and returns an error if it is unable.
func (q *dedupingQueue) push(msg *central.MsgFromSensor) {
	metrics.IncrementSensorEventQueueCounter(ops.Add, common.GetMessageType(msg))
	q.mutex.Lock()
	defer q.mutex.Unlock()

	defer q.notEmptySig.Signal()
	q.pushNoLock(msg)
}

func (q *dedupingQueue) pushNoLock(msg *central.MsgFromSensor) {
	if msg.GetDedupeKey() == "" {
		q.queue.PushBack(msg)
		return
	}
	var msgInserted *list.Element
	if evt, ok := q.resourceIDToEvent[msg.GetDedupeKey()]; ok {
		metrics.IncrementSensorEventQueueCounter(ops.Dedupe, common.GetMessageType(msg))
		msgInserted = q.queue.InsertBefore(msg, evt)
		q.queue.Remove(evt)
	} else {
		msgInserted = q.queue.PushBack(msg)
	}
	q.resourceIDToEvent[msg.GetDedupeKey()] = msgInserted
}
