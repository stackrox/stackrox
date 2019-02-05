package listener

import (
	"github.com/openshift/client-go/apps/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func handleAllEvents(sif informers.SharedInformerFactory, osf externalversions.SharedInformerFactory, output chan<- *central.SensorEvent, stopSignal *concurrency.Signal) {
	// We want creates to be treated as updates while existing objects are loaded.
	var treatCreatesAsUpdates concurrency.Flag
	treatCreatesAsUpdates.Set(true)

	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	podInformer := sif.Core().V1().Pods()
	dispatchers := resources.NewDispatcherRegistry(podInformer.Lister(), clusterentities.StoreInstance())

	// Non-deployment types.
	handle(sif.Core().V1().Namespaces().Informer(), dispatchers.ForNamespaces(), output, nil)
	handle(sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), output, nil)
	handle(sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), output, nil)
	handle(sif.Core().V1().Secrets().Informer(), dispatchers.ForSecrets(), output, nil)
	handle(sif.Core().V1().Services().Informer(), dispatchers.ForServices(), output, nil)

	// Deployment types.
	handle(podInformer.Informer(), dispatchers.ForDeployments(kubernetes.Pod), output, &treatCreatesAsUpdates)
	handle(sif.Extensions().V1beta1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetes.DaemonSet), output, &treatCreatesAsUpdates)
	handle(sif.Extensions().V1beta1().Deployments().Informer(), dispatchers.ForDeployments(kubernetes.Deployment), output, &treatCreatesAsUpdates)
	handle(sif.Extensions().V1beta1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetes.ReplicaSet), output, &treatCreatesAsUpdates)
	handle(sif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetes.ReplicationController), output, &treatCreatesAsUpdates)
	handle(sif.Apps().V1beta1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetes.StatefulSet), output, &treatCreatesAsUpdates)
	handle(sif.Batch().V1().Jobs().Informer(), dispatchers.ForDeployments(kubernetes.Job), output, &treatCreatesAsUpdates)
	handle(sif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetes.CronJob), output, &treatCreatesAsUpdates)

	if osf != nil {
		handle(osf.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetes.DeploymentConfig), output, &treatCreatesAsUpdates)
	}

	// Run the pod informer first since other handlers rely on it's output.
	go podInformer.Informer().Run(stopSignal.Done())
	cache.WaitForCacheSync(stopSignal.Done(), podInformer.Informer().HasSynced)

	// Start our informers and wait for the caches of each to sync so that we know all objects that existed at startup
	// have been consumed.
	sif.Start(stopSignal.Done())
	sif.WaitForCacheSync(stopSignal.Done())
	if osf != nil {
		osf.Start(stopSignal.Done())
		osf.WaitForCacheSync(stopSignal.Done())
	}

	// Set the flag that all objects present at start up have been consumed.
	treatCreatesAsUpdates.Set(false)
}

// Helper function that creates and adds a handler to an informer.
//////////////////////////////////////////////////////////////////
func handle(informer cache.SharedIndexInformer, dispatcher resources.Dispatcher, output chan<- *central.SensorEvent, treatCreatesAsUpdates *concurrency.Flag) {
	informer.AddEventHandler(&resourceEventHandlerImpl{
		dispatcher:            dispatcher,
		output:                output,
		treatCreatesAsUpdates: treatCreatesAsUpdates,
	})
}
