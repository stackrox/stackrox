package listener

import (
	"github.com/openshift/client-go/apps/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/roxmetadata"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func handleAllEvents(sif informers.SharedInformerFactory, osf externalversions.SharedInformerFactory, output chan<- *central.SensorEvent, stopSignal *concurrency.Signal) {
	// We want creates to be treated as updates while existing objects are loaded.
	var treatCreatesAsUpdates concurrency.Flag
	treatCreatesAsUpdates.Set(true)

	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	podInformer := sif.Core().V1().Pods()
	dispatchers := resources.NewDispatcherRegistry(podInformer.Lister(), clusterentities.StoreInstance(), roxmetadata.Singleton())

	namespaceInformer := sif.Core().V1().Namespaces().Informer()
	secretInformer := sif.Core().V1().Secrets().Informer()

	wg := &concurrency.WaitGroup{}

	// Non-deployment types.
	handle(namespaceInformer, dispatchers.ForNamespaces(), output, nil, wg, stopSignal)
	handle(sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), output, nil, wg, stopSignal)
	handle(sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), output, nil, wg, stopSignal)
	handle(secretInformer, dispatchers.ForSecrets(), output, nil, wg, stopSignal)
	handle(sif.Core().V1().Services().Informer(), dispatchers.ForServices(), output, nil, wg, stopSignal)
	handle(sif.Core().V1().ServiceAccounts().Informer(), dispatchers.ForServiceAccounts(), output, nil, wg, stopSignal)

	// RBAC dispatchers handles multiple sets of data
	handle(sif.Rbac().V1().Roles().Informer(), dispatchers.ForRoles(), output, nil, wg, stopSignal)
	handle(sif.Rbac().V1().ClusterRoles().Informer(), dispatchers.ForClusterRoles(), output, nil, wg, stopSignal)
	handle(sif.Rbac().V1().RoleBindings().Informer(), dispatchers.ForRoleBindings(), output, nil, wg, stopSignal)
	handle(sif.Rbac().V1().ClusterRoleBindings().Informer(), dispatchers.ForClusterRoleBindings(), output, nil, wg, stopSignal)

	// Deployment types.
	handle(podInformer.Informer(), dispatchers.ForDeployments(kubernetes.Pod), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Extensions().V1beta1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetes.DaemonSet), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Extensions().V1beta1().Deployments().Informer(), dispatchers.ForDeployments(kubernetes.Deployment), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Extensions().V1beta1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetes.ReplicaSet), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetes.ReplicationController), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Apps().V1beta1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetes.StatefulSet), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Batch().V1().Jobs().Informer(), dispatchers.ForDeployments(kubernetes.Job), output, &treatCreatesAsUpdates, wg, stopSignal)
	handle(sif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetes.CronJob), output, &treatCreatesAsUpdates, wg, stopSignal)

	if osf != nil {
		handle(osf.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetes.DeploymentConfig), output, &treatCreatesAsUpdates, wg, stopSignal)
	}

	// Run the pod and namespace informers first since other handlers rely on their outputs.
	informersToSync := []cache.SharedInformer{podInformer.Informer(), namespaceInformer, secretInformer}
	syncFuncs := make([]cache.InformerSynced, len(informersToSync))
	for i, informer := range informersToSync {
		go informer.Run(stopSignal.Done())
		syncFuncs[i] = informer.HasSynced
	}
	cache.WaitForCacheSync(stopSignal.Done(), syncFuncs...)

	// Start our informers and wait for the caches of each to sync so that we know all objects that existed at startup
	// have been consumed.
	sif.Start(stopSignal.Done())
	if osf != nil {
		osf.Start(stopSignal.Done())
	}
	// WaitForCacheSync synchronization is broken for SharedIndexInformers due to internal addCh/pendingNotifications
	// copy.  We have implemented our own sync in order to work around this.

	if !concurrency.WaitInContext(wg, stopSignal) {
		return
	}

	// Set the flag that all objects present at start up have been consumed.
	treatCreatesAsUpdates.Set(false)

	output <- &central.SensorEvent{
		Resource: &central.SensorEvent_Synced{
			Synced: &central.SensorEvent_ResourcesSynced{},
		},
	}
}

// Helper function that creates and adds a handler to an informer.
//////////////////////////////////////////////////////////////////
func handle(informer cache.SharedIndexInformer, dispatcher resources.Dispatcher, output chan<- *central.SensorEvent, treatCreatesAsUpdates *concurrency.Flag, wg *concurrency.WaitGroup, stopSignal *concurrency.Signal) {
	handlerImpl := &resourceEventHandlerImpl{
		dispatcher:            dispatcher,
		output:                output,
		treatCreatesAsUpdates: treatCreatesAsUpdates,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
	informer.AddEventHandler(handlerImpl)
	wg.Add(1)
	go func() {
		defer wg.Add(-1)
		if !cache.WaitForCacheSync(stopSignal.Done(), informer.HasSynced) {
			return
		}
		doneChannel := handlerImpl.PopulateInitialObjects(informer.GetIndexer().List())
		select {
		case <-stopSignal.Done():
		case <-doneChannel:
		}
	}()
}
