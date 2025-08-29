package listener

import (
	"context"

	osAppsExtVersions "github.com/openshift/client-go/apps/informers/externalversions"
	osConfigExtVersions "github.com/openshift/client-go/config/informers/externalversions"
	osOperatorExtVersions "github.com/openshift/client-go/operator/informers/externalversions"
	osRouteExtVersions "github.com/openshift/client-go/route/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	listenerUtils "github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	complianceOperatorAvailabilityChecker "github.com/stackrox/rox/sensor/kubernetes/listener/watcher/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/crd"
	virtualMachineAvailabilityChecker "github.com/stackrox/rox/sensor/kubernetes/listener/watcher/virtualmachine"
	sensorUtils "github.com/stackrox/rox/sensor/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

type startable interface {
	Start(stopCh <-chan struct{})
}

func startAndWait(stopSignal concurrency.Waitable, wg *concurrency.WaitGroup, startables ...startable) bool {
	for _, start := range startables {
		if start == nil {
			continue
		}
		start.Start(stopSignal.Done())
	}
	return concurrency.WaitInContext(wg, stopSignal)
}

func managedFieldsTransformer(obj interface{}) (interface{}, error) {
	if obj == nil {
		return obj, nil
	}
	if managedFieldsSetter, ok := obj.(interface{ SetManagedFields([]v1.ManagedFieldsEntry) }); ok {
		// Managed fields are unused by Sensor so clear them out to avoid them hitting the cache
		managedFieldsSetter.SetManagedFields(nil)
	}
	return obj, nil
}

type callbackCondition func(*watcher.Status) bool

func allResourcesAvailable() callbackCondition {
	return func(status *watcher.Status) bool {
		return status.Available
	}
}

func resourcesUnavailable() callbackCondition {
	return func(status *watcher.Status) bool {
		return !status.Available
	}
}

func crdWatcherCallbackWrapper(ctx context.Context, cond callbackCondition, pubSub *internalmessage.MessageSubscriber, text string) crd.WatcherCallback {
	return func(status *watcher.Status) {
		if !cond(status) {
			return
		}
		log.Info(status.String())
		if err := pubSub.Publish(&internalmessage.SensorInternalMessage{
			Kind:     internalmessage.SensorMessageSoftRestart,
			Text:     text,
			Validity: ctx,
		}); err != nil {
			log.Errorf("Unable to publish message %s: %v", internalmessage.SensorMessageSoftRestart, err)
		}
	}
}

// handleAllEvents starts the dispatchers for all the kubernetes resources
// tracked by Sensor. For each dispatcher, we wait until it is fully synced,
// meaning we received and processed the initial resources from the cluster.
//
// This is a synchronous process and can be time-consuming. Sensor is in an
// unready state until all resource dispatchers finish syncing, so keep them
// quick. Also, the go-client documentation recommends swift processing, see
//
//	https://github.com/kubernetes/client-go/blob/592d891671b2a09e5f81781b28ebe078d8115e41/tools/cache/shared_informer.go#L128-L132).
//
// We did some stress testing to determine what would happen if a dispatcher is
// blocked for long periods of time. The results indicated that Sensor recovers
// eventually, but we should not take this as a valid reason to disregard the
// previous paragraph because the tests were focused on the startup of secrets
// and there are many unknowns regarding other resources. Plus, having this
// function take long is not ideal since other components of Sensor rely on the
// dispatchers to be synced, see e.g.
//
//	https://github.com/stackrox/stackrox/pull/11662)
//
// The order in the startup process is important since some resources depend on
// others. For example, the pod's informer needs to sync before the deployment's
// since the PodLister is used to populate the image ids of deployments.
func (k *listenerImpl) handleAllEvents() {
	defer k.mayCreateHandlers.Signal()
	sif := informers.NewSharedInformerFactory(k.client.Kubernetes(), noResyncPeriod)
	crdSharedInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod)
	dynamicSif := dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod)
	concurrency.WithLock(&k.sifLock, func() {
		k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, sif)
		k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, crdSharedInformerFactory)
		k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, dynamicSif)
	})

	// Create informer factories for needed orchestrators.
	var osAppsFactory osAppsExtVersions.SharedInformerFactory
	if k.client.OpenshiftApps() != nil {
		osAppsFactory = osAppsExtVersions.NewSharedInformerFactory(k.client.OpenshiftApps(), noResyncPeriod)
		concurrency.WithLock(&k.sifLock, func() {
			k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, osAppsFactory)
		})
	}

	var osRouteFactory osRouteExtVersions.SharedInformerFactory
	if k.client.OpenshiftRoute() != nil {
		osRouteFactory = osRouteExtVersions.NewSharedInformerFactory(k.client.OpenshiftRoute(), noResyncPeriod)
		concurrency.WithLock(&k.sifLock, func() {
			k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, osRouteFactory)
		})
	}

	// We want creates to be treated as updates while existing objects are loaded.
	var syncingResources concurrency.Flag
	syncingResources.Set(true)

	// This might block if a cluster ID is initially unavailable, which is okay.
	clusterID := clusterid.Get()

	// Compliance Operator Watcher and Informers
	var complianceResultInformer, complianceProfileInformer, complianceTailoredProfileInformer, complianceScanSettingBindingsInformer, complianceRuleInformer, complianceScanInformer, complianceSuiteInformer, complianceRemediationInformer cache.SharedIndexInformer
	var profileLister cache.GenericLister

	coCrdWatcher := crd.NewCRDWatcher(&k.stopSig, dynamicSif)
	coAvailabilityChecker := complianceOperatorAvailabilityChecker.NewComplianceOperatorAvailabilityChecker()
	if err := coAvailabilityChecker.AppendToCRDWatcher(coCrdWatcher); err != nil {
		log.Errorf("Unable to add the Resource to the Compliance Operator CRD Watcher: %v", err)
	}

	coCrdHandlerFn := crdWatcherCallbackWrapper(k.context,
		allResourcesAvailable(),
		k.pubSub,
		"Compliance Operator resources have been updated. Connection will restart to force reconciliation with Central",
	)

	// Any informer created in the following block should be added to the coAvailabilityChecker
	coAvailable, err := coAvailabilityChecker.Available(k.client)
	if err != nil {
		log.Errorf("Failed to check the availability of Compliance Operator resources: %v", err)
	}
	if coAvailable {
		log.Info("Initializing compliance operator informers")
		complianceResultInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceCheckResult.GroupVersionResource()).Informer()
		complianceProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Informer()
		profileLister = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Lister()

		complianceScanSettingBindingsInformer = crdSharedInformerFactory.ForResource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Informer()
		complianceRuleInformer = crdSharedInformerFactory.ForResource(complianceoperator.Rule.GroupVersionResource()).Informer()
		complianceScanInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceScan.GroupVersionResource()).Informer()
		complianceTailoredProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.TailoredProfile.GroupVersionResource()).Informer()
		complianceSuiteInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceSuite.GroupVersionResource()).Informer()
		complianceRemediationInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceRemediation.GroupVersionResource()).Informer()
		// Override the coCrdHandlerFn to only handle when the resources become unavailable
		coCrdHandlerFn = crdWatcherCallbackWrapper(k.context,
			resourcesUnavailable(),
			k.pubSub,
			"Compliance Operator resources have been removed. Connection will restart to force reconciliation with Central",
		)
	}

	if err := coCrdWatcher.Watch(coCrdHandlerFn); err != nil {
		log.Errorf("Failed to start watching the Compliance Operator CRDs: %v", err)
	}

	// VirtualMachine Watcher
	vmWatcher := crd.NewCRDWatcher(&k.stopSig, dynamicSif)
	vmAvailabilityChecker := virtualMachineAvailabilityChecker.NewAvailabilityChecker()
	if err := vmAvailabilityChecker.AppendToCRDWatcher(vmWatcher); err != nil {
		log.Errorf("Unable to add the Resource to the VirtualMachine CRD Watcher: %v", err)
	}

	vmCrdHandlerFn := crdWatcherCallbackWrapper(k.context,
		allResourcesAvailable(),
		k.pubSub,
		"VirtualMachine resources have been updated. Connection will restart to force reconciliation with Central")

	virtualMachineIsAvailable, err := vmAvailabilityChecker.Available(k.client)
	if err != nil {
		log.Errorf("Failed to check the availability of Virtual Machine resources: %v", err)
	}
	if virtualMachineIsAvailable {
		// Override the vmCrdHandlerFn to only handle when the resources become unavailable
		vmCrdHandlerFn = crdWatcherCallbackWrapper(k.context,
			resourcesUnavailable(),
			k.pubSub,
			"VirtualMachine resources have been removed. Connection will restart to force reconciliation with Central")
	}
	if err := vmWatcher.Watch(vmCrdHandlerFn); err != nil {
		log.Errorf("Failed to start watching the VirtualMachine CRDs: %v", err)
	}

	// This call to clusterID.Get might block if a cluster ID is initially unavailable, which is okay.
	clusterID := k.clusterID.Get()

	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	podInformer := sif.Core().V1().Pods()
	dispatchers := resources.NewDispatcherRegistry(
		clusterID,
		podInformer.Lister(),
		profileLister,
		processfilter.Singleton(),
		k.configHandler,
		k.credentialsManager,
		k.traceWriter,
		k.storeProvider,
		k.client.Kubernetes(),
	)

	namespaceInformer := sif.Core().V1().Namespaces().Informer()
	secretInformer := sif.Core().V1().Secrets().Informer()
	saInformer := sif.Core().V1().ServiceAccounts().Informer()

	roleInformer := sif.Rbac().V1().Roles().Informer()
	clusterRoleInformer := sif.Rbac().V1().ClusterRoles().Informer()

	// The group that has no other object dependencies
	noDependencyWaitGroup := &concurrency.WaitGroup{}

	// we will single-thread event processing using this lock
	var eventLock sync.Mutex
	stopSignal := &k.stopSig

	// Informers that need to be synced initially
	handle(k.context, namespaceInformer, dispatchers.ForNamespaces(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(k.context, secretInformer, dispatchers.ForSecrets(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(k.context, saInformer, dispatchers.ForServiceAccounts(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)

	// Roles need to be synced before role bindings because role bindings have a reference
	handle(k.context, roleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(k.context, clusterRoleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)

	// For openshift clusters only
	var osConfigFactory osConfigExtVersions.SharedInformerFactory
	if k.client.OpenshiftConfig() != nil {
		if resourceList, err := listenerUtils.ServerResourcesForGroup(k.client, osConfigGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osConfigGroupVersion, err)
		} else {
			osConfigFactory = osConfigExtVersions.NewSharedInformerFactory(k.client.OpenshiftConfig(), noResyncPeriod)
			concurrency.WithLock(&k.sifLock, func() {
				k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, osConfigFactory)
			})

			if listenerUtils.ResourceExists(resourceList, osClusterOperatorsResourceName, osConfigGroupVersion) {
				log.Infof("Initializing %q informer", osClusterOperatorsResourceName)
				handle(k.context, osConfigFactory.Config().V1().ClusterOperators().Informer(), dispatchers.ForClusterOperators(), k.outputQueue, nil, noDependencyWaitGroup, stopSignal, &eventLock)
			}

			if env.RegistryMirroringEnabled.BooleanSetting() {
				if listenerUtils.ResourceExists(resourceList, osImageDigestMirrorSetsResourceName, osConfigGroupVersion) {
					log.Infof("Initializing %q informer", osImageDigestMirrorSetsResourceName)
					handle(k.context, osConfigFactory.Config().V1().ImageDigestMirrorSets().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
				}

				if listenerUtils.ResourceExists(resourceList, osImageTagMirrorSetsResourceName, osConfigGroupVersion) {
					log.Infof("Initializing %q informer", osImageTagMirrorSetsResourceName)
					handle(k.context, osConfigFactory.Config().V1().ImageTagMirrorSets().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
				}
			}
		}
	}

	var osOperatorFactory osOperatorExtVersions.SharedInformerFactory
	if k.client.OpenshiftOperator() != nil && env.RegistryMirroringEnabled.BooleanSetting() {
		if resourceList, err := listenerUtils.ServerResourcesForGroup(k.client, osOperatorAlphaGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osOperatorAlphaGroupVersion, err)
		} else {
			osOperatorFactory = osOperatorExtVersions.NewSharedInformerFactory(k.client.OpenshiftOperator(), noResyncPeriod)
			concurrency.WithLock(&k.sifLock, func() {
				k.sharedInformersToShutdown = append(k.sharedInformersToShutdown, osOperatorFactory)
			})

			if listenerUtils.ResourceExists(resourceList, osImageContentSourcePoliciesResourceName, osOperatorAlphaGroupVersion) {
				log.Infof("Initializing %q informer", osImageContentSourcePoliciesResourceName)
				handle(k.context, osOperatorFactory.Operator().V1alpha1().ImageContentSourcePolicies().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
			}
		}
	}

	if coAvailable {
		log.Info("Syncing compliance operator resources")
		// Handle results, rules, and scan setting bindings first
		handle(k.context, complianceResultInformer, dispatchers.ForComplianceOperatorResults(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(k.context, complianceRuleInformer, dispatchers.ForComplianceOperatorRules(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(k.context, complianceScanSettingBindingsInformer, dispatchers.ForComplianceOperatorScanSettingBindings(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(k.context, complianceScanInformer, dispatchers.ForComplianceOperatorScans(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(k.context, complianceSuiteInformer, dispatchers.ForComplianceOperatorSuites(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(k.context, complianceRemediationInformer, dispatchers.ForComplianceOperatorRemediations(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	}

	if !startAndWait(stopSignal, noDependencyWaitGroup, sif, osConfigFactory, osOperatorFactory, crdSharedInformerFactory) {
		return
	}
	log.Info("Successfully synced secrets, service accounts and roles")

	// prePodWaitGroup
	prePodWaitGroup := &concurrency.WaitGroup{}

	roleBindingInformer := sif.Rbac().V1().RoleBindings().Informer()
	clusterRoleBindingInformer := sif.Rbac().V1().ClusterRoleBindings().Informer()

	handle(k.context, roleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock)
	handle(k.context, clusterRoleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock)

	if !startAndWait(stopSignal, prePodWaitGroup, sif) {
		return
	}

	log.Info("Successfully synced role bindings")

	// Wait for the pod informer to sync before processing other types.
	// This is required because the PodLister is used to populate the image ids of deployments.
	// However, do not ACTUALLY handle, pod events yet -- those need to wait for deployments to be
	// synced, since we need to enrich pods with the deployment ids, and for that we need the entire
	// hierarchy to be populated.
	if !cache.WaitForCacheSync(stopSignal.Done(), podInformer.Informer().HasSynced) {
		return
	}
	log.Info("Successfully synced k8s pod cache")

	preTopLevelDeploymentWaitGroup := &concurrency.WaitGroup{}

	// Non-deployment types.
	handle(k.context, sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(k.context, sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(k.context, sif.Core().V1().Services().Informer(), dispatchers.ForServices(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	if osRouteFactory != nil {
		handle(k.context, osRouteFactory.Route().V1().Routes().Informer(), dispatchers.ForOpenshiftRoutes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}

	// Deployment subtypes (this ensures that the hierarchy maps are generated correctly)
	handle(k.context, sif.Batch().V1().Jobs().Informer(), dispatchers.ForJobs(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(k.context, sif.Apps().V1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicaSet), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(k.context, sif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicationController), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	// Compliance operator profiles are handled AFTER results, rules, and scan setting bindings have been synced
	if complianceProfileInformer != nil {
		handle(k.context, complianceProfileInformer, dispatchers.ForComplianceOperatorProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}
	if complianceTailoredProfileInformer != nil {
		handle(k.context, complianceTailoredProfileInformer, dispatchers.ForComplianceOperatorTailoredProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}

	if !startAndWait(stopSignal, preTopLevelDeploymentWaitGroup, sif, crdSharedInformerFactory, osRouteFactory) {
		return
	}

	log.Info("Successfully synced network policies, nodes, services, jobs, replica sets, and replication controllers")

	wg := &concurrency.WaitGroup{}

	// Deployment types.
	handle(k.context, sif.Apps().V1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.DaemonSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	handle(k.context, sif.Apps().V1().Deployments().Informer(), dispatchers.ForDeployments(kubernetesPkg.Deployment), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	handle(k.context, sif.Apps().V1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.StatefulSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)

	if ok, err := sensorUtils.HasAPI(k.client.Kubernetes(), "batch/v1", kubernetesPkg.CronJob); err != nil {
		log.Errorf("error determining API version to use for CronJobs: %v", err)
	} else if ok {
		handle(k.context, sif.Batch().V1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	} else {
		handle(k.context, sif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	}
	if osAppsFactory != nil {
		handle(k.context, osAppsFactory.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetesPkg.DeploymentConfig), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	}

	// SharedInformerFactories can have Start called multiple times which will start the rest of the handlers
	if !startAndWait(stopSignal, wg, sif, osAppsFactory) {
		return
	}

	log.Info("Successfully synced daemonsets, deployments, stateful sets and cronjobs")

	// Finally, run the pod informer, and process pod events.
	podWaitGroup := &concurrency.WaitGroup{}
	handle(k.context, podInformer.Informer(), dispatchers.ForDeployments(kubernetesPkg.Pod), k.outputQueue, &syncingResources, podWaitGroup, stopSignal, &eventLock)
	if !startAndWait(stopSignal, podWaitGroup, sif) {
		return
	}

	log.Info("Successfully synced pods")

	// Set the flag that all objects present at start up have been consumed.
	syncingResources.Set(false)

	k.outputQueue.Send(&component.ResourceEvent{
		ForwardMessages: []*central.SensorEvent{
			{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
		Context: k.context,
	})
	utils.Should(k.pubSub.Publish(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageResourceSyncFinished,
		Text:     "Finished the k8s resource sync",
		Validity: k.context,
	}))
}

// Helper function that creates and adds a handler to an informer.
// ////////////////////////////////////////////////////////////////
func handle(
	ctx context.Context,
	informer cache.SharedIndexInformer,
	dispatcher resources.Dispatcher,
	resolver component.Resolver,
	syncingResources *concurrency.Flag,
	wg *concurrency.WaitGroup,
	stopSignal *concurrency.Signal,
	eventLock *sync.Mutex,
) {
	handlerImpl := &resourceEventHandlerImpl{
		context:          ctx,
		eventLock:        eventLock,
		dispatcher:       dispatcher,
		resolver:         resolver,
		syncingResources: syncingResources,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
	_, err := informer.AddEventHandler(handlerImpl)
	should(err, stopSignal)
	if !informer.HasSynced() {
		err := informer.SetTransform(managedFieldsTransformer)
		should(err, stopSignal)
	}
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

// should function wraps utils.Should to avoid panics if the listeners were already stopped by Sensor.
func should(err error, stopSignal *concurrency.Signal) {
	if err == nil {
		return
	}
	// We don't want to panic in development builds if adding a handler fails due to the listener being stopped.
	if stopSignal.IsDone() {
		log.Warnf("Error while the informers were stopped: %+v", err)
		return
	}
	utils.Should(err)
}
