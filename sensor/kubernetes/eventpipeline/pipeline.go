package eventpipeline

import (
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
)

func New(client client.Interface, configHandler config.Handler, detector detector.Detector, nodeName string, resyncPeriod time.Duration, traceWriter io.Writer) common.SensorComponent {
	stopSig := concurrency.NewSignal()
	outputQueue := output.New(&stopSig, detector)
	resolverQueue := resolver.New(outputQueue)

	resourceListener := listener.New(client, configHandler, nodeName, resyncPeriod, traceWriter, resolverQueue)

	pipelineResposnes := make(chan *central.MsgFromSensor)
	return &eventPipeline{
		eventsC:  pipelineResposnes,
		listener: resourceListener,
		stopSig:  &stopSig,
		output:   outputQueue,
	}
}
