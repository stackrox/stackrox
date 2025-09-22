package sensor

import (
	"context"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common"
)

type ComponentProcessor struct {
	queues []*ComponentQueue
}

func NewComponentProcessor(ctx context.Context, receivers []common.SensorComponent) *ComponentProcessor {
	queues := make([]*ComponentQueue, 0, len(receivers))
	for _, c := range receivers {
		componentQueue := NewComponentQueue(c)
		componentQueue.Start(ctx)
		queues = append(queues, componentQueue)
	}

	return &ComponentProcessor{
		queues: queues,
	}
}

func (p *ComponentProcessor) ProcessMessage(msg *central.MsgToSensor) {
	for _, q := range p.queues {
		q.Push(msg)
	}
}