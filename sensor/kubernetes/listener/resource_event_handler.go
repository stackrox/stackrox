package listener

import (
	osAppsExtVersions "github.com/openshift/client-go/apps/informers/externalversions"
	osConfigExtVersions "github.com/openshift/client-go/config/informers/externalversions"
	osRouteExtVersions "github.com/openshift/client-go/route/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
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

func startAndWait(stopSignal *concurrency.Signal, wg *concurrency.WaitGroup, startables ...startable) bool {
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
	// TODO(ROX-14194): remove resyncingSif once all resources are adapted
	var resyncingSif informers.SharedInformerFactory
	if env.ResyncDisabled.BooleanSetting() {
		resyncingSif = informers.NewSharedInformerFactory(k.client.Kubernetes(), noResyncPeriod)
	} else {
		resyncingSif = informers.NewSharedInformerFactory(k.client.Kubernetes(), k.resyncPeriod)
	}
	sif := informers.NewSharedInformerFactory(k.client.Kubernetes(), noResyncPeriod)

	// Create informer factories for needed orchestrators.
	var osAppsFactory osAppsExtVersions.SharedInformerFactory
	if k.client.OpenshiftApps() != nil {
		osAppsFactory = osAppsExtVersions.NewSharedInformerFactory(k.client.OpenshiftApps(), k.resyncPeriod)
	}

	var osRouteFactory osRouteExtVersions.SharedInformerFactory
	if k.client.OpenshiftRoute() != nil {
		osRouteFactory = osRouteExtVersions.NewSharedInformerFactory(k.client.OpenshiftRoute(), k.resyncPeriod)
	}

	// We want creates to be treated as updates while existing objects are loaded.
	var syncingResources concurrency.Flag
	syncingResources.Set(true)

	// This might block if a cluster ID is initially unavailable, which is okay.
	clusterID := clusterid.Get()

	var crdSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	var complianceResultInformer, complianceProfileInformer, complianceTailoredProfileInformer, complianceScanSettingBindingsInformer, complianceRuleInformer, complianceScanInformer cache.SharedIndexInformer
	var profileLister cache.GenericLister
	if ok, err := complianceCRDExists(k.client.Kubernetes()); err != nil {
		log.Errorf("error finding compliance CRD: %v", err)
	} else if !ok {
		log.Info("compliance CRD could not be found")
	} else {
		log.Infof("initializing compliance operator informers")
		crdSharedInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod)
		complianceResultInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceCheckResult.GroupVersionResource()).Informer()
		complianceProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Informer()
		profileLister = crdSharedInformerFactory.ForResource(complianceoperator.Profile.GroupVersionResource()).Lister()

		complianceScanSettingBindingsInformer = crdSharedInformerFactory.ForResource(complianceoperator.ScanSettingBinding.GroupVersionResource()).Informer()
		complianceRuleInformer = crdSharedInformerFactory.ForResource(complianceoperator.Rule.GroupVersionResource()).Informer()
		complianceScanInformer = crdSharedInformerFactory.ForResource(complianceoperator.ComplianceScan.GroupVersionResource()).Informer()
		complianceTailoredProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.TailoredProfile.GroupVersionResource()).Informer()
	}

	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	podInformer := resyncingSif.Core().V1().Pods()
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
	handle(namespaceInformer, dispatchers.ForNamespaces(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(secretInformer, dispatchers.ForSecrets(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(saInformer, dispatchers.ForServiceAccounts(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)

	// Roles need to be synced before role bindings because role bindings have a reference
	handle(roleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	handle(clusterRoleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)

	var osConfigFactory osConfigExtVersions.SharedInformerFactory
	if k.client.OpenshiftConfig() != nil {
		if ok, err := clusterOperatorCRDExists(k.client); err != nil {
			log.Errorf("Error checking for cluster operator CRD: %v", err)
		} else if !ok {
			log.Warnf("Skipping cluster operator discovery....")
		} else {
			osConfigFactory = osConfigExtVersions.NewSharedInformerFactory(k.client.OpenshiftConfig(), noResyncPeriod)
		}
	}
	// For openshift clusters only
	if osConfigFactory != nil {
		handle(osConfigFactory.Config().V1().ClusterOperators().Informer(), dispatchers.ForClusterOperators(),
			k.outputQueue, nil, noDependencyWaitGroup, stopSignal, &eventLock)
	}

	if crdSharedInformerFactory != nil {
		log.Info("syncing compliance operator resources")
		// Handle results, rules, and scan setting bindings first
		handle(complianceResultInformer, dispatchers.ForComplianceOperatorResults(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(complianceRuleInformer, dispatchers.ForComplianceOperatorRules(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(complianceScanSettingBindingsInformer, dispatchers.ForComplianceOperatorScanSettingBindings(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
		handle(complianceScanInformer, dispatchers.ForComplianceOperatorScans(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock)
	}

	if !startAndWait(stopSignal, noDependencyWaitGroup, sif, resyncingSif, osConfigFactory, crdSharedInformerFactory) {
		return
	}
	log.Info("Successfully synced secrets, service accounts and roles")

	// prePodWaitGroup
	prePodWaitGroup := &concurrency.WaitGroup{}

	roleBindingInformer := resyncingSif.Rbac().V1().RoleBindings().Informer()
	clusterRoleBindingInformer := resyncingSif.Rbac().V1().ClusterRoleBindings().Informer()

	handle(roleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock)
	handle(clusterRoleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock)

	if !startAndWait(stopSignal, prePodWaitGroup, resyncingSif) {
		return
	}

	log.Info("Successfully synced role bindings")

	// Wait for the pod informer to sync before processing other types.
	// This is required because the PodLister is used to populate the image ids of deployments.
	// However, do not ACTUALLY handle pod events yet -- those need to wait for deployments to be
	// synced, since we need to enrich pods with the deployment ids, and for that we need the entire
	// hierarchy to be populated.
	if !cache.WaitForCacheSync(stopSignal.Done(), podInformer.Informer().HasSynced) {
		return
	}
	log.Info("Successfully synced k8s pod cache")

	preTopLevelDeploymentWaitGroup := &concurrency.WaitGroup{}

	// Non-deployment types.
	handle(sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(sif.Core().V1().Services().Informer(), dispatchers.ForServices(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	if osRouteFactory != nil {
		handle(osRouteFactory.Route().V1().Routes().Informer(), dispatchers.ForOpenshiftRoutes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}

	// Deployment subtypes (this ensures that the hierarchy maps are generated correctly)
	handle(resyncingSif.Batch().V1().Jobs().Informer(), dispatchers.ForJobs(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicaSet), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(resyncingSif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicationController), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	// Compliance operator profiles are handled AFTER results, rules, and scan setting bindings have been synced
	if complianceProfileInformer != nil {
		handle(complianceProfileInformer, dispatchers.ForComplianceOperatorProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}
	if complianceTailoredProfileInformer != nil {
		handle(complianceTailoredProfileInformer, dispatchers.ForComplianceOperatorTailoredProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}

	if !startAndWait(stopSignal, preTopLevelDeploymentWaitGroup, sif, resyncingSif, crdSharedInformerFactory, osRouteFactory) {
		return
	}

	log.Info("Successfully synced network policies, nodes, services, jobs, replica sets, and replication controllers")

	wg := &concurrency.WaitGroup{}

	// Deployment types.
	handle(resyncingSif.Apps().V1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.DaemonSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().Deployments().Informer(), dispatchers.ForDeployments(kubernetesPkg.Deployment), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.StatefulSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)

	if ok, err := sensorUtils.HasAPI(k.client.Kubernetes(), "batch/v1", kubernetesPkg.CronJob); err != nil {
		log.Errorf("error determining API version to use for CronJobs: %v", err)
	} else if ok {
		handle(resyncingSif.Batch().V1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	} else {
		handle(resyncingSif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	}
	if osAppsFactory != nil {
		handle(osAppsFactory.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetesPkg.DeploymentConfig), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock)
	}

	// SharedInformerFactories can have Start called multiple times which will start the rest of the handlers
	if !startAndWait(stopSignal, wg, sif, resyncingSif, osAppsFactory) {
		return
	}

	log.Info("Successfully synced daemonsets, deployments, stateful sets and cronjobs")

	// Finally, run the pod informer, and process pod events.
	podWaitGroup := &concurrency.WaitGroup{}
	handle(podInformer.Informer(), dispatchers.ForDeployments(kubernetesPkg.Pod), k.outputQueue, &syncingResources, podWaitGroup, stopSignal, &eventLock)
	if !startAndWait(stopSignal, podWaitGroup, resyncingSif) {
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
	})
}

// Helper function that creates and adds a handler to an informer.
// ////////////////////////////////////////////////////////////////
func handle(
	informer cache.SharedIndexInformer,
	dispatcher resources.Dispatcher,
	resolver component.Resolver,
	syncingResources *concurrency.Flag,
	wg *concurrency.WaitGroup,
	stopSignal *concurrency.Signal,
	eventLock *sync.Mutex,
) {
	handlerImpl := &resourceEventHandlerImpl{
		eventLock:        eventLock,
		dispatcher:       dispatcher,
		resolver:         resolver,
		syncingResources: syncingResources,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
	_, err := informer.AddEventHandler(handlerImpl)
	utils.Should(err)
	if !informer.HasSynced() {
		err := informer.SetTransform(managedFieldsTransformer)
		utils.Should(err)
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
