package eventpipeline

import (
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
)

// New instantiates the eventPipeline component
func New(client client.Interface, configHandler config.Handler, detector detector.Detector, nodeName string, resyncPeriod time.Duration, traceWriter io.Writer) common.SensorComponent {
	// TODO(ROX-13413): Move this env.EventPipelineOutputQueueSize to CreateOptions
	outputQueue := output.New(detector, env.EventPipelineOutputQueueSize.IntegerSetting())
	resourceListener := listener.New(client, configHandler, nodeName, resyncPeriod, traceWriter, outputQueue)

	pipelineResponses := make(chan *central.MsgFromSensor)
	return &eventPipeline{
		eventsC:  pipelineResponses,
		stopSig:  concurrency.NewSignal(),
		output:   outputQueue,
		listener: resourceListener,
	}
}
