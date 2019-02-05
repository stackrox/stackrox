package streamer

import (
	"container/list"
	"sync"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	ops "github.com/stackrox/rox/pkg/metrics"
)

type queueImpl struct {
	mutex             sync.Mutex
	notEmptySig       concurrency.Signal
	queue             *list.List
	resourceIDToEvent map[string]*list.Element
}

func newQueue() *queueImpl {
	return &queueImpl{
		notEmptySig:       concurrency.NewSignal(),
		queue:             list.New(),
		resourceIDToEvent: make(map[string]*list.Element),
	}
}

func (p *queueImpl) notEmpty() concurrency.WaitableChan {
	return p.notEmptySig.WaitC()
}

func (p *queueImpl) pull() *central.MsgFromSensor {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.queue.Len() == 0 {
		return nil
	}

	evt := p.queue.Remove(p.queue.Front()).(*central.MsgFromSensor)
	metrics.IncrementSensorEventQueueCounter(ops.Remove, common.GetMessageType(evt))
	// If resource action was not create, then delete it from the cache
	if evt.GetEvent() != nil && evt.GetEvent().GetAction() != central.ResourceAction_CREATE_RESOURCE {
		delete(p.resourceIDToEvent, evt.GetEvent().GetId())
	}

	if p.queue.Len() == 0 {
		p.notEmptySig.Reset()
	}
	return evt
}

// Push attempts to add an item to the queue, and returns an error if it is unable.
func (p *queueImpl) push(msg *central.MsgFromSensor) {
	metrics.IncrementSensorEventQueueCounter(ops.Add, common.GetMessageType(msg))
	p.mutex.Lock()
	defer p.mutex.Unlock()

	defer p.notEmptySig.Signal()
	p.pushNoLock(msg)
}

func (p *queueImpl) pushNoLock(msg *central.MsgFromSensor) {
	if msg.GetEvent().GetAction() == central.ResourceAction_CREATE_RESOURCE || msg.GetEvent() == nil {
		p.queue.PushBack(msg)
		// Purposefully don't cache the CREATE because it should never be deduped
		return
	}
	var msgInserted *list.Element
	if evt, ok := p.resourceIDToEvent[msg.GetEvent().GetId()]; ok {
		metrics.IncrementSensorEventQueueCounter(ops.Dedupe, common.GetMessageType(msg))
		msgInserted = p.queue.InsertBefore(msg, evt)
		p.queue.Remove(evt)
	} else {
		msgInserted = p.queue.PushBack(msg)
	}
	p.resourceIDToEvent[msg.GetEvent().GetId()] = msgInserted
	return
}
