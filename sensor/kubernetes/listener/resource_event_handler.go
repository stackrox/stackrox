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
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	listenerUtils "github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	complianceOperatorAvailabilityChecker "github.com/stackrox/rox/sensor/kubernetes/listener/watcher/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/crd"
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

func (k *listenerImpl) handleAllEvents() {
	defer k.mayCreateHandlers.Signal()
	sif := informers.NewSharedInformerFactory(k.client.Kubernetes(), noResyncPeriod)

	// Create informer factories for needed orchestrators.
	var osAppsFactory osAppsExtVersions.SharedInformerFactory
	if k.client.OpenshiftApps() != nil {
		osAppsFactory = osAppsExtVersions.NewSharedInformerFactory(k.client.OpenshiftApps(), noResyncPeriod)
	}

	var osRouteFactory osRouteExtVersions.SharedInformerFactory
	if k.client.OpenshiftRoute() != nil {
		osRouteFactory = osRouteExtVersions.NewSharedInformerFactory(k.client.OpenshiftRoute(), noResyncPeriod)
	}

	// We want creates to be treated as updates while existing objects are loaded.
	var syncingResources concurrency.Flag
	syncingResources.Set(true)

	// This might block if a cluster ID is initially unavailable, which is okay.
	clusterID := clusterid.Get()

	var crdSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	var complianceResultInformer, complianceProfileInformer, complianceTailoredProfileInformer, complianceScanSettingBindingsInformer, complianceRuleInformer, complianceScanInformer, complianceSuiteInformer, complianceRemediationInformer cache.SharedIndexInformer
	var profileLister cache.GenericLister
	crdWatcher := crd.NewCRDWatcher(&k.stopSig, dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod))
	coAvailabilityChecker := complianceOperatorAvailabilityChecker.NewComplianceOperatorAvailabilityChecker()
	if err := coAvailabilityChecker.AppendToCRDWatcher(crdWatcher); err != nil {
		log.Errorf("Unable to add the Resource to the CRD Watcher: %v", err)
	}
	if err := crdWatcher.Watch(k.crdWatcherStatusC); err != nil {
		log.Errorf("Failed to start watching the CRDs: %v", err)
	}
	crdHandlerFn := func(status *watcher.Status) {
		if status.Available {
			log.Infof("Resources %v became available", status.Resources)
			if err := k.pubSub.Publish(&internalmessage.SensorInternalMessage{
				Kind:     internalmessage.SensorMessageSoftRestart,
				Text:     "Compliance Operator resources have been updated. Connection will restart to force reconciliation with Central",
				Validity: k.context,
			}); err != nil {
				log.Errorf("Unable to publish message %s: %v", internalmessage.SensorMessageSoftRestart, err)
			}
		}
	}
	// Any informer created in the following block should be added to the coAvailabilityChecker
	if coAvailabilityChecker.Available(k.client) {
		log.Info("initializing compliance operator informers")
		crdSharedInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod)
		complianceResultInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceCheckResult.GroupVersionResource()).Informer()
		complianceProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Informer()
		profileLister = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Lister()

		complianceScanSettingBindingsInformer = crdSharedInformerFactory.ForResource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Informer()
		complianceRuleInformer = crdSharedInformerFactory.ForResource(complianceoperator.Rule.GroupVersionResource()).Informer()
		complianceScanInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceScan.GroupVersionResource()).Informer()
		complianceTailoredProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.TailoredProfile.GroupVersionResource()).Informer()
		complianceSuiteInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceSuite.GroupVersionResource()).Informer()
		complianceRemediationInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceRemediation.GroupVersionResource()).Informer()
		// Override the crdHandlerFn to only handle when the resources become unavailable
		crdHandlerFn = func(status *watcher.Status) {
			if !status.Available {
				log.Infof("Resources %v became unavailable", status.Resources)
				if err := k.pubSub.Publish(&internalmessage.SensorInternalMessage{
					Kind:     internalmessage.SensorMessageSoftRestart,
					Text:     "Compliance Operator resources have been removed. Connection will restart to force reconciliation with Central",
					Validity: k.context,
				}); err != nil {
					log.Errorf("Unable to publish message %s: %v", internalmessage.SensorMessageSoftRestart, err)
				}
			}
		}
	}
	k.handleWatcherStatus(crdHandlerFn)

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

			if listenerUtils.ResourceExists(resourceList, osImageContentSourcePoliciesResourceName, osOperatorAlphaGroupVersion) {
				log.Infof("Initializing %q informer", osImageContentSourcePoliciesResourceName)
				handle(k.context, osOperatorFactory.Operator().V1alpha1().ImageContentSourcePolicies().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
			}
		}
	}

	if crdSharedInformerFactory != nil {
		log.Info("syncing compliance operator resources")
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
