package eventpipeline

import (
	"io"
	"sync/atomic"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/common/message"
	"github.com/stackrox/rox/sensor/common/reprocessor"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/output"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/resolver"
	"github.com/stackrox/rox/sensor/kubernetes/listener"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
)

// New instantiates the eventPipeline component
func New(client client.Interface, configHandler config.Handler, detector detector.Detector, reprocessor reprocessor.Handler, nodeName string, resyncPeriod time.Duration, traceWriter io.Writer, storeProvider *resources.StoreProvider, queueSize int) common.SensorComponent {
	outputQueue := output.New(detector, queueSize)
	var depResolver component.Resolver
	var resourceListener component.ContextListener
	depResolver = resolver.New(outputQueue, storeProvider, queueSize)
	resourceListener = listener.New(client, configHandler, nodeName, resyncPeriod, traceWriter, depResolver, storeProvider)

	offlineMode := &atomic.Bool{}
	offlineMode.Store(true)

	pipelineResponses := make(chan *message.ExpiringMessage)
	return &eventPipeline{
		eventsC:     pipelineResponses,
		stopSig:     concurrency.NewSignal(),
		output:      outputQueue,
		resolver:    depResolver,
		listener:    resourceListener,
		detector:    detector,
		reprocessor: reprocessor,
		offlineMode: offlineMode,
	}
}
