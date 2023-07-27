package listener

import (
	"io"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/sensor/common/awscredentials"
	"github.com/stackrox/rox/sensor/common/config"
	"github.com/stackrox/rox/sensor/kubernetes/client"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
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
	stopSig            concurrency.Signal
	credentialsManager awscredentials.RegistryCredentialsManager
	configHandler      config.Handler
	resyncPeriod       time.Duration
	traceWriter        io.Writer
	outputQueue        component.Resolver
	storeProvider      *resources.InMemoryStoreProvider
}

func (k *listenerImpl) Start() error {
	// This happens if the listener is restarting. Then the signal will already have been triggered
	// when starting a new run of the listener.
	if k.stopSig.IsDone() {
		k.stopSig.Reset()
	}

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
