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
	var complianceResultInformer, complianceProfileInformer, complianceTailoredProfileInformer, complianceScanSettingBindingsInformer, complianceRuleInformer, complianceScanInformer, complianceSuiteInformer cache.SharedIndexInformer
	var profileLister cache.GenericLister
	if resourceList, err := serverResourcesForGroup(k.client, complianceoperator.GetGroupVersion().String()); err != nil {
		log.Errorf("Checking API resources for group %q: %v", complianceoperator.GetGroupVersion().String(), err)
	} else if resourceExists(resourceList, complianceoperator.ComplianceCheckResult.Name) {
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
	}

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
	handle(k.context, namespaceInformer, dispatchers.ForNamespaces(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "namespace")

	// The wait group for secrets
	secretsWaitGroup := &concurrency.WaitGroup{}
	// setup the secrets handler
	handle(k.context, secretInformer, dispatchers.ForSecrets(), k.outputQueue, &syncingResources, secretsWaitGroup, stopSignal, &eventLock, "secret")
	// start just the secrets informer
	if !startAndWait(stopSignal, secretsWaitGroup, sif) {
		log.Info("ROX-24163: startAndWait failed for secrets")
	}

	handle(k.context, saInformer, dispatchers.ForServiceAccounts(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "sa")
	// Roles need to be synced before role bindings because role bindings have a reference
	handle(k.context, roleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "role")
	handle(k.context, clusterRoleInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "clusterrole")

	// For openshift clusters only
	var osConfigFactory osConfigExtVersions.SharedInformerFactory
	if k.client.OpenshiftConfig() != nil {
		if resourceList, err := serverResourcesForGroup(k.client, osConfigGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osConfigGroupVersion, err)
		} else {
			osConfigFactory = osConfigExtVersions.NewSharedInformerFactory(k.client.OpenshiftConfig(), noResyncPeriod)

			if resourceExists(resourceList, osClusterOperatorsResourceName) {
				log.Infof("Initializing %q informer", osClusterOperatorsResourceName)
				handle(k.context, osConfigFactory.Config().V1().ClusterOperators().Informer(), dispatchers.ForClusterOperators(), k.outputQueue, nil, noDependencyWaitGroup, stopSignal, &eventLock, "clusteroperator")
			}

			if env.RegistryMirroringEnabled.BooleanSetting() {
				if resourceExists(resourceList, osImageDigestMirrorSetsResourceName) {
					log.Infof("Initializing %q informer", osImageDigestMirrorSetsResourceName)
					handle(k.context, osConfigFactory.Config().V1().ImageDigestMirrorSets().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "registrymirror-digest")
				}

				if resourceExists(resourceList, osImageTagMirrorSetsResourceName) {
					log.Infof("Initializing %q informer", osImageTagMirrorSetsResourceName)
					handle(k.context, osConfigFactory.Config().V1().ImageTagMirrorSets().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "registrymirror-tag")
				}
			}
		}
	}

	var osOperatorFactory osOperatorExtVersions.SharedInformerFactory
	if k.client.OpenshiftOperator() != nil && env.RegistryMirroringEnabled.BooleanSetting() {
		if resourceList, err := serverResourcesForGroup(k.client, osOperatorAlphaGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osOperatorAlphaGroupVersion, err)
		} else {
			osOperatorFactory = osOperatorExtVersions.NewSharedInformerFactory(k.client.OpenshiftOperator(), noResyncPeriod)

			if resourceExists(resourceList, osImageContentSourcePoliciesResourceName) {
				log.Infof("Initializing %q informer", osImageContentSourcePoliciesResourceName)
				handle(k.context, osOperatorFactory.Operator().V1alpha1().ImageContentSourcePolicies().Informer(), dispatchers.ForRegistryMirrors(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "imagecontentsourcepolicy")
			}
		}
	}

	if crdSharedInformerFactory != nil {
		log.Info("Syncing compliance operator resources")
		// Handle results, rules, and scan setting bindings first
		handle(k.context, complianceResultInformer, dispatchers.ForComplianceOperatorResults(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "coResults")
		handle(k.context, complianceRuleInformer, dispatchers.ForComplianceOperatorRules(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "coRules")
		handle(k.context, complianceScanSettingBindingsInformer, dispatchers.ForComplianceOperatorScanSettingBindings(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "coScanSettingBindings")
		handle(k.context, complianceScanInformer, dispatchers.ForComplianceOperatorScans(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "coScans")
		handle(k.context, complianceSuiteInformer, dispatchers.ForComplianceOperatorSuites(), k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, "coSuites")
	}

	log.Debug("ROX-24163: Waiting for sync of: namespaces, secrets, service accounts, roles, and cluster roles")
	if !startAndWait(stopSignal, noDependencyWaitGroup, sif, osConfigFactory, osOperatorFactory, crdSharedInformerFactory) {
		log.Debug("ROX-24163: startAndWait failed for: namespaces, secrets, service accounts, roles, and cluster roles")
		return
	}
	log.Info("Successfully synced namespaces, secrets, service accounts, roles, and cluster roles")

	// prePodWaitGroup
	prePodWaitGroup := &concurrency.WaitGroup{}

	roleBindingInformer := sif.Rbac().V1().RoleBindings().Informer()
	clusterRoleBindingInformer := sif.Rbac().V1().ClusterRoleBindings().Informer()

	handle(k.context, roleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock, "rolebinding")
	handle(k.context, clusterRoleBindingInformer, dispatchers.ForRBAC(), k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock, "clusterrolebinding")

	if !startAndWait(stopSignal, prePodWaitGroup, sif) {
		log.Debug("ROX-24163: startAndWait failed for: rolebindings and clusterrole binding")
		return
	}

	log.Info("Successfully synced role bindings")

	// Wait for the pod informer to sync before processing other types.
	// This is required because the PodLister is used to populate the image ids of deployments.
	// However, do not ACTUALLY handle, pod events yet -- those need to wait for deployments to be
	// synced, since we need to enrich pods with the deployment ids, and for that we need the entire
	// hierarchy to be populated.
	log.Debug("ROX-24163 Waiting for pod Informer to sync")
	if !cache.WaitForCacheSync(stopSignal.Done(), podInformer.Informer().HasSynced) {
		log.Debug("ROX-24163: startAndWait failed for pods")
		return
	}
	log.Info("Successfully synced k8s pod cache")

	preTopLevelDeploymentWaitGroup := &concurrency.WaitGroup{}

	// Non-deployment types.
	handle(k.context, sif.Networking().V1().NetworkPolicies().Informer(), dispatchers.ForNetworkPolicies(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "networkpolicy")
	handle(k.context, sif.Core().V1().Nodes().Informer(), dispatchers.ForNodes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "node")
	handle(k.context, sif.Core().V1().Services().Informer(), dispatchers.ForServices(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "svc")

	if osRouteFactory != nil {
		handle(k.context, osRouteFactory.Route().V1().Routes().Informer(), dispatchers.ForOpenshiftRoutes(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "route")
	}

	// Deployment subtypes (this ensures that the hierarchy maps are generated correctly)
	handle(k.context, sif.Batch().V1().Jobs().Informer(), dispatchers.ForJobs(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "job")
	handle(k.context, sif.Apps().V1().ReplicaSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicaSet), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "replicaset")
	handle(k.context, sif.Core().V1().ReplicationControllers().Informer(), dispatchers.ForDeployments(kubernetesPkg.ReplicationController), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "replicationcontroller")

	// Compliance operator profiles are handled AFTER results, rules, and scan setting bindings have been synced
	if complianceProfileInformer != nil {
		handle(k.context, complianceProfileInformer, dispatchers.ForComplianceOperatorProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "coProfiles")
	}
	if complianceTailoredProfileInformer != nil {
		handle(k.context, complianceTailoredProfileInformer, dispatchers.ForComplianceOperatorTailoredProfiles(), k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, "coTailoredProfiles")
	}

	if !startAndWait(stopSignal, preTopLevelDeploymentWaitGroup, sif, crdSharedInformerFactory, osRouteFactory) {
		return
	}

	log.Info("Successfully synced network policies, nodes, services, jobs, replica sets, and replication controllers")

	wg := &concurrency.WaitGroup{}

	// Deployment types.
	handle(k.context, sif.Apps().V1().DaemonSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.DaemonSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "daemonset")
	handle(k.context, sif.Apps().V1().Deployments().Informer(), dispatchers.ForDeployments(kubernetesPkg.Deployment), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "deployment")
	handle(k.context, sif.Apps().V1().StatefulSets().Informer(), dispatchers.ForDeployments(kubernetesPkg.StatefulSet), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "statefulset")

	if ok, err := sensorUtils.HasAPI(k.client.Kubernetes(), "batch/v1", kubernetesPkg.CronJob); err != nil {
		log.Errorf("error determining API version to use for CronJobs: %v", err)
	} else if ok {
		handle(k.context, sif.Batch().V1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "cronjobs")
	} else {
		handle(k.context, sif.Batch().V1beta1().CronJobs().Informer(), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "cronjobs-v1beta")
	}
	if osAppsFactory != nil {
		handle(k.context, osAppsFactory.Apps().V1().DeploymentConfigs().Informer(), dispatchers.ForDeployments(kubernetesPkg.DeploymentConfig), k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, "apps")
	}

	// SharedInformerFactories can have Start called multiple times which will start the rest of the handlers
	if !startAndWait(stopSignal, wg, sif, osAppsFactory) {
		return
	}

	log.Info("Successfully synced daemonsets, deployments, stateful sets and cronjobs")

	// Finally, run the pod informer, and process pod events.
	podWaitGroup := &concurrency.WaitGroup{}
	handle(k.context, podInformer.Informer(), dispatchers.ForDeployments(kubernetesPkg.Pod), k.outputQueue, &syncingResources, podWaitGroup, stopSignal, &eventLock, "pod")
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
	name string,
) {
	handlerImpl := &resourceEventHandlerImpl{
		name:             name,
		context:          ctx,
		eventLock:        eventLock,
		dispatcher:       dispatcher,
		resolver:         resolver,
		syncingResources: syncingResources,

		hasSeenAllInitialIDsSignal: concurrency.NewSignal(),
		seenIDs:                    make(map[types.UID]struct{}),
		missingInitialIDs:          nil,
	}
	handlerRegistration, err := informer.AddEventHandler(handlerImpl)
	should(err, stopSignal)

	log.Debugf("ROX-24163 (start) for %q has synced: hReg=%t informer=%t", name, handlerRegistration.HasSynced(), informer.HasSynced())

	if !informer.HasSynced() {
		log.Debugf("ROX-24163 Informer for %q has not synced. Applying transformation", name)
		err := informer.SetTransform(managedFieldsTransformer)
		should(err, stopSignal)
	}
	wg.Add(1)
	go func() {
		defer func() {
			wg.Add(-1)
			log.Debugf("ROX-24163 (defer) for %q has synced: hReg=%t informer=%t", name, handlerRegistration.HasSynced(), informer.HasSynced())
		}()
		log.Debugf("ROX-24163 (go func) for %q has synced: hReg=%t informer=%t", name, handlerRegistration.HasSynced(), informer.HasSynced())
		if !cache.WaitForCacheSync(stopSignal.Done(), informer.HasSynced) {
			return
		}
		log.Debugf("ROX-24163 calling PopulateInitialObjects for %q", name)
		doneChannel := handlerImpl.PopulateInitialObjects(informer.GetIndexer().List())

		select {
		case <-stopSignal.Done():
			log.Debugf("ROX-24163 handle for %q received stop signal", name)
		case <-doneChannel:
			log.Debugf("ROX-24163 done populating InitialObjects for %q", name)
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
