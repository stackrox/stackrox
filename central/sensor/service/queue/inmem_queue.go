package queue

import (
	"container/list"
	"sync"

	"github.com/stackrox/rox/central/metrics"
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

	metrics.IncrementSensorEventQueueCounter(ops.Remove)
	evt := p.queue.Remove(p.queue.Front()).(*central.MsgFromSensor)
	delete(p.resourceIDToEvent, evt.GetEvent().GetId())

	return evt, nil
}

// Push attempts to add an item to the queue, and returns an error if it is unable.
func (p *queueImpl) Push(msg *central.MsgFromSensor) error {
	metrics.IncrementSensorEventQueueCounter(ops.Add)
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Dedupe event messages that are not deployments. Don't dedupe deployments, it prevents enforcement from
	// happening when we get a CREATE then an UPDATE in the queue.
	if msg.GetEvent() != nil && msg.GetEvent().GetDeployment() == nil {
		return p.pushWithDedupeNoLock(msg)
	}
	return p.pushNoLock(msg)
}

func (p *queueImpl) pushNoLock(msg *central.MsgFromSensor) error {
	p.queue.PushBack(msg)
	return nil
}

func (p *queueImpl) pushWithDedupeNoLock(msg *central.MsgFromSensor) error {
	if evt, ok := p.resourceIDToEvent[msg.GetEvent().GetId()]; ok {
		metrics.IncrementSensorEventQueueCounter(ops.Dedupe)
		p.queue.Remove(evt)
	}
	msgInserted := p.queue.PushBack(msg)
	p.resourceIDToEvent[msg.GetEvent().GetId()] = msgInserted
	return nil
}
