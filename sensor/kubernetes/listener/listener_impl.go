package listener

import (
	"time"

	"github.com/openshift/client-go/apps/informers/externalversions"
	configExtVersions "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"k8s.io/client-go/informers"
)

const (
	// See https://groups.google.com/forum/#!topic/kubernetes-sig-api-machinery/PbSCXdLDno0
	// Kubernetes scheduler no longer uses a resync period and it seems like its usage doesn't apply to us
	resyncPeriod                = 0
	resyncingPeriod             = 1 * time.Minute
	clusterOperatorResourceName = "clusteroperators"
	clusterOperatorGroupVersion = "config.openshift.io/v1"
)

type listenerImpl struct {
	client  client.Interface
	eventsC chan *central.MsgFromSensor
	stopSig concurrency.Signal

	configHandler config.Handler
	detector      detector.Detector
}

func (k *listenerImpl) Start() error {
	// Create informer factories for needed orchestrators.
	var osAppsFactory externalversions.SharedInformerFactory
	var osConfigFactory configExtVersions.SharedInformerFactory

	k8sFactory := informers.NewSharedInformerFactory(k.client.Kubernetes(), resyncPeriod)
	k8sResyncingFactory := informers.NewSharedInformerFactory(k.client.Kubernetes(), resyncingPeriod)
	if k.client.OpenshiftApps() != nil {
		osAppsFactory = externalversions.NewSharedInformerFactory(k.client.OpenshiftApps(), resyncingPeriod)
	}
	if k.client.OpenshiftConfig() != nil {
		ok, err := clusterOperatorCRDExists(k.client)
		if ok && err == nil {
			osConfigFactory = configExtVersions.NewSharedInformerFactory(k.client.OpenshiftConfig(), resyncingPeriod)
		} else {
			if err != nil {
				log.Errorf("Error checking for cluster operator CRD: %v", err)
			}
			log.Warnf("Skipping cluster operator discovery....")
		}
	}

	// Patch namespaces to include labels
	patchNamespaces(k.client.Kubernetes(), &k.stopSig)

	// Start handling resource events.
	go handleAllEvents(k8sFactory, k8sResyncingFactory, osAppsFactory, osConfigFactory, k.eventsC, &k.stopSig, k.configHandler, k.detector)
	return nil
}

func (k *listenerImpl) Stop(_ error) {
	k.stopSig.Signal()
}

func (k *listenerImpl) Capabilities() []centralsensor.SensorCapability {
	return nil
}

func (k *listenerImpl) ProcessMessage(_ *central.MsgToSensor) error {
	return nil
}

func (k *listenerImpl) ResponsesC() <-chan *central.MsgFromSensor {
	return k.eventsC
}

func clusterOperatorCRDExists(client client.Interface) (bool, error) {
	resourceList, err := client.Kubernetes().Discovery().ServerResourcesForGroupVersion(clusterOperatorGroupVersion)
	if err != nil {
		return false, err
	}
	for _, apiResource := range resourceList.APIResources {
		if apiResource.Name == clusterOperatorResourceName {
			return true, nil
		}
	}
	return false, nil
}
