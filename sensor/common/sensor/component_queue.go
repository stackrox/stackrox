package sensor

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type ComponentQueue struct {
	component common.SensorComponent
	q         *queue.Queue[*central.MsgToSensor]
}

func NewComponentQueue(component common.SensorComponent) *ComponentQueue {
	c := &ComponentQueue{
		component: component,
		q: queue.NewQueue(
			queue.WithQueueName[*central.MsgToSensor](component.Name()),
			queue.WithMaxSize[*central.MsgToSensor](env.RequestsChannelBufferSize.IntegerSetting()),
			queue.WithCounterVec[*central.MsgToSensor](metrics.ComponentQueueOperations),
			queue.WithDroppedMetric[*central.MsgToSensor](metrics.ComponentQueueMessagesDroppedCount),
		),
	}
	return c
}

func (c ComponentQueue) Push(msg *central.MsgToSensor) {
	c.q.Push(msg)
}

func (c ComponentQueue) Start(ctx context.Context) {
	go c.start(ctx)
}

func (c ComponentQueue) start(stopCtx context.Context) {
	for msg := range c.q.Seq(stopCtx) {
		start := time.Now()
		processCtx, cancelFunc := context.WithTimeout(stopCtx, time.Second)
		if err := c.component.ProcessMessage(processCtx, msg); err != nil {
			log.Errorf("%s.ProcessMessage(%q) errored: %v", c.component.Name(), msg.String(), err)
		}
		cancelFunc()
		metrics.ObserveCentralReceiverProcessMessageDuration(c.component.Name(), time.Since(start))
	}
}
