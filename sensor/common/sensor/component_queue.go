package sensor

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/env"
	op "github.com/stackrox/rox/pkg/metrics"
	"github.com/stackrox/rox/pkg/queue"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/metrics"
)

const (
	// componentProcessTimeout is the maximum time allowed for a sensor component to process a message from Central
	componentProcessTimeout = 30 * time.Second
)

type ComponentQueue struct {
	component common.SensorComponent
	q         *queue.Queue[*central.MsgToSensor]
}

func NewComponentQueue(component common.SensorComponent) *ComponentQueue {
	componentName := prometheus.Labels{
		metrics.ComponentName: component.Name(),
	}
	componentQueueOperations, err := metrics.ComponentQueueOperations.CurryWith(componentName)
	utils.CrashOnError(err)
	c := &ComponentQueue{
		component: component,
		q: queue.NewQueue(
			queue.WithQueueName[*central.MsgToSensor](component.Name()),
			queue.WithMaxSize[*central.MsgToSensor](env.RequestsChannelBufferSize.IntegerSetting()),
			queue.WithCounterVec[*central.MsgToSensor](componentQueueOperations),
			queue.WithDroppedMetric[*central.MsgToSensor](componentQueueOperations.With(prometheus.Labels{
				metrics.Operation: op.Dropped.String(),
			})),
		),
	}
	return c
}

func (c ComponentQueue) Push(msg *central.MsgToSensor) {
	if !c.component.Filter(msg) {
		return
	}
	c.q.Push(msg)
}

func (c ComponentQueue) Start(ctx context.Context) {
	go c.start(ctx)
}

func (c ComponentQueue) start(stopCtx context.Context) {
	for msg := range c.q.Seq(stopCtx) {
		start := time.Now()
		processCtx, cancelFunc := context.WithTimeout(stopCtx, componentProcessTimeout)
		if err := c.component.ProcessMessage(processCtx, msg); err != nil {
			log.Errorf("%s.ProcessMessage(%q) errored: %v", c.component.Name(), msg.String(), err)
			metrics.IncrementCentralReceiverProcessMessageErrors(c.component.Name())
		}
		cancelFunc()
		metrics.ObserveCentralReceiverProcessMessageDuration(c.component.Name(), time.Since(start))
	}
}
