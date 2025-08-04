package eventpipeline

import (
	"io"
	"sync/atomic"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

// New instantiates the eventPipeline component. Sensors pipeline is responsible for the entire lifecycle of a Kubernetes
// event. From receiving it from the listeners, converting it to storage.Deployment, finding related resources to be
// reprocessed, and sending events directly to Central or to the detector for processing.
//
// Components will communicate with each other using component.ResourceEvent data structure. The pipeline is organized as:
//
//	Listener -> Resolver -> Output
//
// Each component will write and consume different properties from ResourceEvent, and send the event to the next component in the chain.
// For an explanation what each property means, check the documentation for component.ResourceEvent.
//
// This component introduces a new type of component to sensor:
// - The event pipeline is a sensor component. That means, it can send messages to the gRPC stream via the .ResponseC function.
// - Pipeline components are sub-components inside the event pipeline that process a kubernetes event from start to finish (listener, resolver and output are all pipeline components)
func New(clusterIDGetter clusterIDGetter, client client.Interface, configHandler config.Handler, detector detector.Detector, reprocessor reprocessor.Handler, nodeName string, traceWriter io.Writer, storeProvider *resources.StoreProvider, queueSize int, pubSub *internalmessage.MessageSubscriber) common.SensorComponent {
	outputQueue := output.New(detector, queueSize)
	depResolver := resolver.New(outputQueue, storeProvider, queueSize)
	resourceListener := listener.New(clusterIDGetter, client, configHandler, nodeName, traceWriter, depResolver, storeProvider, pubSub)

	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)

	pipelineResponses := make(chan *message.ExpiringMessage)
	return &eventPipeline{
		eventsC:     pipelineResponses,
		stopper:     concurrency.NewStopper(),
		output:      outputQueue,
		resolver:    depResolver,
		listener:    resourceListener,
		detector:    detector,
		reprocessor: reprocessor,
		offlineMode: offlineMode,
	}
}

type clusterIDGetter interface {
	Get() string
}
