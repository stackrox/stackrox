package queue

import (
	"container/list"
	"sync"

	"github.com/stackrox/rox/central/metrics"
	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/generated/internalapi/central"
	ops "github.com/stackrox/rox/pkg/metrics"
)

// Queue provides an interface for a queue that stores MsgFromSensor.
type Queue interface {
	Push(sensor *central.MsgFromSensor) error
	Pull() (*central.MsgFromSensor, error)
}

type queueImpl struct {
	mutex sync.Mutex

	queue             *list.List
	resourceIDToEvent map[string]*list.Element
}

// NewQueue initializes a SensorEvent queue
func NewQueue() Queue {
	return &queueImpl{
		queue:             list.New(),
		resourceIDToEvent: make(map[string]*list.Element),
	}
}

// Pull attempts to get the next item from the queue.
// When no items are available, both the event and error will be nil.
func (p *queueImpl) Pull() (*central.MsgFromSensor, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if p.queue.Len() == 0 {
		return nil, nil
	}

	evt := p.queue.Remove(p.queue.Front()).(*central.MsgFromSensor)
	metrics.IncrementSensorEventQueueCounter(ops.Remove, common.GetMessageType(evt))
	// If resource action was not create, then delete it from the cache
	if evt.GetEvent().GetAction() != central.ResourceAction_CREATE_RESOURCE {
		delete(p.resourceIDToEvent, evt.GetEvent().GetId())
	}

	return evt, nil
}

// Push attempts to add an item to the queue, and returns an error if it is unable.
func (p *queueImpl) Push(msg *central.MsgFromSensor) error {
	metrics.IncrementSensorEventQueueCounter(ops.Add, common.GetMessageType(msg))
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.pushNoLock(msg)
}

func (p *queueImpl) pushNoLock(msg *central.MsgFromSensor) error {
	if msg.GetEvent().GetAction() == central.ResourceAction_CREATE_RESOURCE {
		p.queue.PushBack(msg)
		// Purposefully don't cache the CREATE because it should never be deduped
		return nil
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
	return nil
}
