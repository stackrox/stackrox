package sensor

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/metrics"
)

type Processor interface {
	Process(ctx context.Context, msg *central.MsgToSensor)
}

type componentsProcessor struct {
	receivers []common.SensorComponent
}

func NewProcessor(receivers ...common.SensorComponent) *componentsProcessor {
	p := &componentsProcessor{
		receivers: receivers,
	}
	return p
}

func (p *componentsProcessor) Process(ctx context.Context, msg *central.MsgToSensor) {
	for _, r := range p.receivers {
		start := time.Now()
		if err := r.ProcessMessage(ctx, msg); err != nil {
			log.Error(err)
		}
		metrics.ObserveCentralReceiverProcessMessageDuration(r.Name(), time.Since(start))
	}
}
