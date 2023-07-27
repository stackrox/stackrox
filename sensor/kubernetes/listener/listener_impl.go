package listener

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/stackrox/rox/pkg/buildinfo"
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
	mayCreateHandlers  concurrency.Signal
	context            context.Context
}

func (k *listenerImpl) StartWithContext(ctx context.Context) error {
	// There is a caveat here that we need to make sure that the previous Start has already
	// finished before swap the context, otherwise there is a risk of data racing by swapping
	// the context while its being used to create the handlers.
	// Since the handleAllEvents function takes too long to run, using mutex is a problem in
	// dev environments, since the mutex will be locked to more than 5s, resulting in a panic.
	// The current workaround is to use a signal instead of a mutex.
	k.mayCreateHandlers.Wait()
	k.mayCreateHandlers.Reset()
	k.context = ctx
	return k.Start()
}

func (k *listenerImpl) Start() error {
	if k.context == nil {
		if !buildinfo.ReleaseBuild {
			panic("Something went very wrong: starting Kubernetes Listener with nil context")
		}
		return errors.New("cannot start listener without a context")
	}

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
	k.storeProvider.CleanupStores()
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
