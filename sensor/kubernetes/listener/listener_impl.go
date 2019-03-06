package listener

import (
	"github.com/openshift/client-go/apps/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/client-go/informers"
)

const (
	// See https://groups.google.com/forum/#!topic/kubernetes-sig-api-machinery/PbSCXdLDno0
	// Kubernetes scheduler no longer uses a resync period and it seems like its usage doesn't apply to us
	resyncPeriod = 0
)

type listenerImpl struct {
	clients *clientSet
	eventsC chan *central.SensorEvent
	stopSig concurrency.Signal
}

func (k *listenerImpl) Start() {
	// Create informer factories for needed orchestrators.
	var k8sFactory informers.SharedInformerFactory
	var osFactory externalversions.SharedInformerFactory
	k8sFactory = informers.NewSharedInformerFactory(k.clients.k8s, resyncPeriod)
	if k.clients.openshift != nil {
		osFactory = externalversions.NewSharedInformerFactory(k.clients.openshift, resyncPeriod)
	}

	// Patch namespaces to include labels
	patchNamespaces(k.clients.k8s, &k.stopSig)

	// Start handling resource events.
	handleAllEvents(k8sFactory, osFactory, k.eventsC, &k.stopSig)
}

func (k *listenerImpl) Stop() {
	k.stopSig.Signal()
}

func (k *listenerImpl) Events() <-chan *central.SensorEvent {
	return k.eventsC
}
