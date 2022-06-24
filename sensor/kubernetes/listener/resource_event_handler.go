package listener

import (
	osAppsExtVersions "github.com/openshift/client-go/apps/informers/externalversions"
	osConfigExtVersions "github.com/openshift/client-go/config/informers/externalversions"
	osRouteExtVersions "github.com/openshift/client-go/route/informers/externalversions"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/common/clusterid"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stackrox/rox/sensor/kubernetes/orchestratornamespaces"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/tools/cache"
)

func (k *listenerImpl) handleAllEvents() {
	sif := informers.NewSharedInformerFactory(k.client.Kubernetes(), noResyncPeriod)
	resyncingSif := informers.NewSharedInformerFactory(k.client.Kubernetes(), k.resyncPeriod)

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
	var treatCreatesAsUpdates concurrency.Flag
	treatCreatesAsUpdates.Set(true)

	// This might block if a cluster ID is initially unavailable, which is okay.
	clusterID := clusterid.Get()

	var crdSharedInformerFactory dynamicinformer.DynamicSharedInformerFactory
	var complianceResultInformer, complianceProfileInformer, complianceTailoredProfileInformer, complianceScanSettingBindingsInformer, complianceRuleInformer, complianceScanInformer cache.SharedIndexInformer
	var profileLister cache.GenericLister
	if features.ComplianceOperatorCheckResults.Enabled() {
		if ok, err := complianceCRDExists(k.client.Kubernetes()); err != nil {
			log.Errorf("error finding compliance CRD: %v", err)
		} else if !ok {
			log.Info("compliance CRD could not be found")
		} else {
			log.Infof("initializing compliance operator informers")
			crdSharedInformerFactory = dynamicinformer.NewDynamicSharedInformerFactory(k.client.Dynamic(), noResyncPeriod)
			complianceResultInformer = crdSharedInformerFactory.ForResource(complianceoperator.CheckResultGVR).Informer()
			complianceProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.ProfileGVR).Informer()
			profileLister = crdSharedInformerFactory.ForResource(complianceoperator.ProfileGVR).Lister()

			complianceScanSettingBindingsInformer = crdSharedInformerFactory.ForResource(complianceoperator.ScanSettingBindingGVR).Informer()
			complianceRuleInformer = crdSharedInformerFactory.ForResource(complianceoperator.RuleGVR).Informer()
			complianceScanInformer = crdSharedInformerFactory.ForResource(complianceoperator.ScanGVR).Informer()
			complianceTailoredProfileInformer = crdSharedInformerFactory.ForResource(complianceoperator.TailoredProfileGVR).Informer()
		}
	}
	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	podInformer := resyncingSif.Core().V1().Pods()
	dispatchers := resources.NewDispatcherRegistry(
		clusterID,
		podInformer.Lister(),
		profileLister,
		clusterentities.StoreInstance(),
		processfilter.Singleton(),
		k.configHandler,
		k.detector,
		orchestratornamespaces.Singleton(),
		k.credentialsManager,
		k.traceWriter,
	)

	namespaceInformer := sif.Core().V1().Namespaces().Informer()
	secretInformer := sif.Core().V1().Secrets().Informer()
	saInformer := sif.Core().V1().ServiceAccounts().Informer()

	roleInformer := sif.Rbac().V1().Roles().Informer()
	clusterRoleInformer := sif.Rbac().V1().ClusterRoles().Informer()
	roleBindingInformer := resyncingSif.Rbac().V1().RoleBindings().Informer()
	clusterRoleBindingInformer := resyncingSif.Rbac().V1().ClusterRoleBindings().Informer()

	// prePodWaitGroup
	prePodWaitGroup := &concurrency.WaitGroup{}

	// we will single-thread event processing using this lock
	var eventLock sync.Mutex
	stopSignal := &k.stopSig

	// Informers that need to be synced initially
	handle(namespaceInformer, dispatchers.ForNamespaces(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	handle(secretInformer, dispatchers.ForSecrets(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	handle(saInformer, dispatchers.ForServiceAccounts(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)

	// RBAC dispatchers handles multiple sets of data
	handle(roleInformer, dispatchers.ForRBAC(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	handle(clusterRoleInformer, dispatchers.ForRBAC(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	handle(roleBindingInformer, dispatchers.ForRBAC(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	handle(clusterRoleBindingInformer, dispatchers.ForRBAC(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)

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
			k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	}

	if crdSharedInformerFactory != nil {
		log.Info("syncing compliance operator resources")
		// Handle results, rules, and scan setting bindings first
		handle(complianceResultInformer, dispatchers.ForComplianceOperatorResults(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
		handle(complianceRuleInformer, dispatchers.ForComplianceOperatorRules(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
		handle(complianceScanSettingBindingsInformer, dispatchers.ForComplianceOperatorScanSettingBindings(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
		handle(complianceScanInformer, dispatchers.ForComplianceOperatorScans(), k.eventsC, nil, prePodWaitGroup, stopSignal, &eventLock)
	}

	sif.Start(stopSignal.Done())
	resyncingSif.Start(stopSignal.Done())
	if osConfigFactory != nil {
		osConfigFactory.Start(stopSignal.Done())
	}
	if crdSharedInformerFactory != nil {
		crdSharedInformerFactory.Start(stopSignal.Done())
	}

	if !concurrency.WaitInContext(prePodWaitGroup, stopSignal) {
		return
	}
	log.Info("Successfully synced namespaces, secrets, service accounts, roles and role bindings")

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
	handle(sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(sif.Core().V1().Services().Informer(), dispatchers.ForServices(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	if osRouteFactory != nil {
		handle(osRouteFactory.Route().V1().Routes().Informer(), dispatchers.ForOpenshiftRoutes(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	}

	// Deployment subtypes (this ensures that the hierarchy maps are generated correctly)
	handle(resyncingSif.Batch().V1().Jobs().Informer(), dispatchers.ForJobs(), k.eventsC, &treatCreatesAsUpdates, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetes.ReplicaSet), k.eventsC, &treatCreatesAsUpdates, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
	handle(resyncingSif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetes.ReplicationController), k.eventsC, &treatCreatesAsUpdates, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)

	if features.ComplianceOperatorCheckResults.Enabled() {
		// Compliance operator profiles are handled AFTER results, rules, and scan setting bindings have been synced
		if complianceProfileInformer != nil {
			handle(complianceProfileInformer, dispatchers.ForComplianceOperatorProfiles(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
		}
		if complianceTailoredProfileInformer != nil {
			handle(complianceTailoredProfileInformer, dispatchers.ForComplianceOperatorTailoredProfiles(), k.eventsC, nil, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock)
		}
	}

	sif.Start(stopSignal.Done())
	resyncingSif.Start(stopSignal.Done())
	if crdSharedInformerFactory != nil {
		crdSharedInformerFactory.Start(stopSignal.Done())
	}
	if osRouteFactory != nil {
		osRouteFactory.Start(stopSignal.Done())
	}

	if !concurrency.WaitInContext(preTopLevelDeploymentWaitGroup, stopSignal) {
		return
	}

	log.Info("Successfully synced network policies, nodes, services, jobs, replica sets, and replication controllers")

	wg := &concurrency.WaitGroup{}

	// Deployment types.
	handle(resyncingSif.Apps().V1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetes.DaemonSet), k.eventsC, &treatCreatesAsUpdates, wg, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().Deployments().Informer(), dispatchers.ForDeployments(kubernetes.Deployment), k.eventsC, &treatCreatesAsUpdates, wg, stopSignal, &eventLock)
	handle(resyncingSif.Apps().V1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetes.StatefulSet), k.eventsC, &treatCreatesAsUpdates, wg, stopSignal, &eventLock)
	handle(resyncingSif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetes.CronJob), k.eventsC, &treatCreatesAsUpdates, wg, stopSignal, &eventLock)

	if osAppsFactory != nil {
		handle(osAppsFactory.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetes.DeploymentConfig), k.eventsC, &treatCreatesAsUpdates, wg, stopSignal, &eventLock)
	}

	// SharedInformerFactories can have Start called multiple times which will start the rest of the handlers
	sif.Start(stopSignal.Done())
	resyncingSif.Start(stopSignal.Done())
	if osAppsFactory != nil {
		osAppsFactory.Start(stopSignal.Done())
	}

	// WaitForCacheSync synchronization is broken for SharedIndexInformers due to internal addCh/pendingNotifications
	// copy.  We have implemented our own sync in order to work around this.
	if !concurrency.WaitInContext(wg, stopSignal) {
		return
	}

	log.Info("Successfully synced daemonsets, deployments, stateful sets and cronjobs")

	// Finally, run the pod informer, and process pod events.
	podWaitGroup := &concurrency.WaitGroup{}
	handle(podInformer.Informer(), dispatchers.ForDeployments(kubernetes.Pod), k.eventsC, &treatCreatesAsUpdates, podWaitGroup, stopSignal, &eventLock)
	sif.Start(stopSignal.Done())

	if !concurrency.WaitInContext(podWaitGroup, stopSignal) {
		return
	}

	log.Info("Successfully synced pods")

	// Set the flag that all objects present at start up have been consumed.
	treatCreatesAsUpdates.Set(false)

	k.eventsC <- &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
	}
}

// Helper function that creates and adds a handler to an informer.
//////////////////////////////////////////////////////////////////
func handle(
	informer cache.SharedIndexInformer,
	dispatcher resources.Dispatcher,
	output chan<- *central.MsgFromSensor,
	treatCreatesAsUpdates *concurrency.Flag,
	wg *concurrency.WaitGroup,
	stopSignal *concurrency.Signal,
	eventLock *sync.Mutex,
) {
	handlerImpl := &resourceEventHandlerImpl{
		eventLock:             eventLock,
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
