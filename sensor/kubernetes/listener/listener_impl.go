package listener

import (
	"io"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/common/detector"
	"github.com/stackrox/rox/sensor/kubernetes/client"
)

const (
	// See https://groups.google.com/forum/#!topic/kubernetes-sig-api-machinery/PbSCXdLDno0
	// Kubernetes scheduler no longer uses a resync period and it seems like its usage doesn't apply to us
	noResyncPeriod              = 0
	clusterOperatorResourceName = "clusteroperators"
	clusterOperatorGroupVersion = "config.openshift.io/v1"
)

type listenerImpl struct {
	client             client.Interface
	eventsC            chan *central.MsgFromSensor
	stopSig            concurrency.Signal
	credentialsManager awscredentials.RegistryCredentialsManager
	configHandler      config.Handler
	detector           detector.Detector
	resyncPeriod       time.Duration
	traceWriter        io.Writer
}

func (k *listenerImpl) Start() error {
	// Patch namespaces to include labels
	patchNamespaces(k.client.Kubernetes(), &k.stopSig)
	// Start credentials manager.
	if k.credentialsManager != nil {
		k.credentialsManager.Start()
	}
	// Start handling resource events.
	go k.handleAllEvents()
	return nil
}

func (k *listenerImpl) Stop(_ error) {
	if k.credentialsManager != nil {
		k.credentialsManager.Stop()
	}
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
