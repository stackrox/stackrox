package eventpipeline

import (
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

// New instantiates the eventPipeline component
func New(client client.Interface, configHandler config.Handler, detector detector.Detector, nodeName string, resyncPeriod time.Duration, traceWriter io.Writer, storeProvider *resources.InMemoryStoreProvider) common.SensorComponent {
	// TODO(ROX-13413): Move this env.EventPipelineOutputQueueSize to CreateOptions
	outputQueue := output.New(detector, env.EventPipelineOutputQueueSize.IntegerSetting())
	var depResolver component.Resolver
	var resourceListener component.PipelineComponent
	if features.ResyncDisabled.Enabled() {
		depResolver = resolver.New(outputQueue, resources.DeploymentStoreSingleton(), storeProvider)
		resourceListener = listener.New(client, configHandler, nodeName, resyncPeriod, traceWriter, depResolver, storeProvider)
	} else {
		resourceListener = listener.New(client, configHandler, nodeName, resyncPeriod, traceWriter, outputQueue, storeProvider)
	}

	pipelineResponses := make(chan *central.MsgFromSensor)
	return &eventPipeline{
		eventsC:  pipelineResponses,
		stopSig:  concurrency.NewSignal(),
		output:   outputQueue,
		resolver: depResolver,
		listener: resourceListener,
	}
}
