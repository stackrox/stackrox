package listener

import (
	"time"

	"github.com/openshift/client-go/apps/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"k8s.io/client-go/informers"
)

const (
	// See https://groups.google.com/forum/#!topic/kubernetes-sig-api-machinery/PbSCXdLDno0
	// Kubernetes scheduler no longer uses a resync period and it seems like its usage doesn't apply to us
	resyncPeriod    = 0
	resyncingPeriod = 1 * time.Minute
)

type listenerImpl struct {
	clients *clientSet
	eventsC chan *central.MsgFromSensor
	stopSig concurrency.Signal

	configHandler config.Handler
	detector      detector.Detector
}

func (k *listenerImpl) Start() error {
	// Create informer factories for needed orchestrators.
	var osFactory externalversions.SharedInformerFactory

	k8sFactory := informers.NewSharedInformerFactoryWithOptions(k.clients.k8s, resyncPeriod)
	k8sResyncingFactory := informers.NewSharedInformerFactory(k.clients.k8s, resyncingPeriod)
	if k.clients.openshift != nil {
		osFactory = externalversions.NewSharedInformerFactory(k.clients.openshift, resyncingPeriod)
	}

	// Patch namespaces to include labels
	patchNamespaces(k.clients.k8s, &k.stopSig)

	// Start handling resource events.
	go handleAllEvents(k8sFactory, k8sResyncingFactory, osFactory, k.eventsC, &k.stopSig, k.configHandler, k.detector)
	return nil
}

func (k *listenerImpl) Stop(err error) {
	k.stopSig.Signal()
}

func (k *listenerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (k *listenerImpl) ProcessMessage(msg *central.MsgToSensor) error {
	return nil
}

func (k *listenerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return k.eventsC
}
