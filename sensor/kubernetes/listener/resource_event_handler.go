package listener

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8swatch"
	kubernetesPkg "github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/common/internalmessage"
	sensorMetrics "github.com/stackrox/rox/sensor/common/metrics"
	"github.com/stackrox/rox/sensor/common/processfilter"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	listenerUtils "github.com/stackrox/rox/sensor/kubernetes/listener/utils"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher"
	complianceOperatorAvailabilityChecker "github.com/stackrox/rox/sensor/kubernetes/listener/watcher/complianceoperator"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watcher/crd"
	virtualMachineAvailabilityChecker "github.com/stackrox/rox/sensor/kubernetes/listener/watcher/virtualmachine"
	sensorUtils "github.com/stackrox/rox/sensor/utils"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	v1Listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

// OpenShift GVRs for dynamic informers. Using dynamic informers instead of
// typed OpenShift client-go informers avoids importing 4 OpenShift scheme
// packages that register ~8400 types at init(), consuming ~10 MB RSS even
// on vanilla k8s clusters that never use OpenShift APIs.
var (
	deploymentConfigGVR     = schema.GroupVersionResource{Group: "apps.openshift.io", Version: "v1", Resource: "deploymentconfigs"}
	routeGVR                = schema.GroupVersionResource{Group: "route.openshift.io", Version: "v1", Resource: "routes"}
	clusterOperatorGVR      = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "clusteroperators"}
	imageDigestMirrorSetGVR = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "imagedigestmirrorsets"}
	imageTagMirrorSetGVR    = schema.GroupVersionResource{Group: "config.openshift.io", Version: "v1", Resource: "imagetagmirrorsets"}
	imageContentSourceGVR   = schema.GroupVersionResource{Group: "operator.openshift.io", Version: "v1alpha1", Resource: "imagecontentsourcepolicies"}
)

// gvrToAPIPath converts a GroupVersionResource to a Kubernetes API path.
// Core resources (no group) use /api/v1, others use /apis/<group>/<version>.
func gvrToAPIPath(gvr schema.GroupVersionResource) string {
	if gvr.Group == "" {
		return fmt.Sprintf("/api/%s/%s", gvr.Version, gvr.Resource)
	}
	return fmt.Sprintf("/apis/%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
}

// noOpGenericLister is a stub lister that satisfies cache.GenericLister but doesn't cache objects.
// Used when k8swatch adapters replace dynamic informers but legacy code still expects a lister.
type noOpGenericLister struct{}

func (n *noOpGenericLister) List(_ labels.Selector) ([]runtime.Object, error) {
	return nil, errors.New("lister not supported for k8swatch adapters")
}

func (n *noOpGenericLister) Get(_ string) (runtime.Object, error) {
	return nil, errors.New("lister not supported for k8swatch adapters")
}

func (n *noOpGenericLister) ByNamespace(_ string) cache.GenericNamespaceLister {
	return &noOpGenericNamespaceLister{}
}

type noOpGenericNamespaceLister struct{}

func (n *noOpGenericNamespaceLister) List(_ labels.Selector) ([]runtime.Object, error) {
	return nil, errors.New("lister not supported for k8swatch adapters")
}

func (n *noOpGenericNamespaceLister) Get(_ string) (runtime.Object, error) {
	return nil, errors.New("lister not supported for k8swatch adapters")
}

func waitForWaitGroup(stopSignal concurrency.Waitable, wg *concurrency.WaitGroup) bool {
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

	var informerTracker *informerSyncTracker
	if features.SensorInformerWatchdog.Enabled() {
		// One log message every 30 seconds if any informers are still pending.
		// This is meant to spam the logs if informers are stuck, as there were many cases
		// where informers were stuck for hours without any indication to the user.
		loggingPeriod := 30 * time.Second
		informerTracker = newInformerSyncTracker(loggingPeriod)
		defer informerTracker.stop()
	}

	// k8swatch adapters are used instead of dynamic shared informer factories.
	// This completely removes dependency on client-go/informers and client-go/kubernetes,
	// eliminating 232 packages from the dependency tree.
	k8sClient := k8swatch.InClusterClient()

	isOpenShift := env.OpenshiftAPI.BooleanSetting()

	// We want creates to be treated as updates while existing objects are loaded.
	var syncingResources concurrency.Flag
	syncingResources.Set(true)

	// Compliance Operator Watcher and Informers
	var (
		complianceResultInformer              cache.SharedIndexInformer
		complianceScanSettingBindingsInformer cache.SharedIndexInformer
		complianceRuleInformer                cache.SharedIndexInformer
		complianceScanInformer                cache.SharedIndexInformer
		complianceSuiteInformer               cache.SharedIndexInformer
		complianceRemediationInformer         cache.SharedIndexInformer
		complianceCustomRuleInformer          cache.SharedIndexInformer
	)
	var profileLister cache.GenericLister

	coCrdWatcher := crd.NewCRDWatcher(&k.stopSig, k8sClient)
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

	var customRulesAvailable bool
	if coAvailable {
		log.Info("Initializing compliance operator informers")
		complianceResultInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.ComplianceCheckResult.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		complianceScanSettingBindingsInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.ScanSettingBinding.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		complianceRuleInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.Rule.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		complianceScanInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.ComplianceScan.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		complianceSuiteInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.ComplianceSuite.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		complianceRemediationInformer = k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.ComplianceRemediation.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)

		customRulesAvailable, err = sensorUtils.HasAPI(k.client.Discovery(), complianceoperator.GetGroupVersion().String(), complianceoperator.CustomRule.Kind)
		if err != nil {
			log.Errorf("Failed to check the availability of Compliance Operator Custom Rules, they won't be tracked: %v", err)
		}
		if customRulesAvailable {
			complianceCustomRuleInformer = k8swatch.NewInformerAdapter(
				gvrToAPIPath(complianceoperator.CustomRule.GroupVersionResource()),
				k8sClient,
				func() runtime.Object { return &unstructured.Unstructured{} },
			)
		}

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

	// VirtualMachine Watcher and Informers
	// We should track virtual machines only if the feature is enabled and CRDs are available.
	shouldTrackVirtualMachines := features.VirtualMachines.Enabled()
	var virtualMachineInstanceInformer cache.SharedIndexInformer

	// Leaving this check explicitely here for clarity, that we don't want to
	// call this code when the feature is disabled.
	if features.VirtualMachines.Enabled() {
		vmWatcher := crd.NewCRDWatcher(&k.stopSig, k8sClient)
		vmAvailabilityChecker := virtualMachineAvailabilityChecker.NewAvailabilityChecker()
		if err := vmAvailabilityChecker.AppendToCRDWatcher(vmWatcher); err != nil {
			log.Errorf("Unable to add the Resource to the VirtualMachine CRD Watcher: %v", err)
		}

		vmCrdHandlerFn := crdWatcherCallbackWrapper(k.context,
			allResourcesAvailable(),
			k.pubSub,
			"VirtualMachine resources have been updated. Connection will restart to force reconciliation with Central")

		shouldTrackVirtualMachines, err = vmAvailabilityChecker.Available(k.client)
		if err != nil {
			log.Errorf("Failed to check the availability of Virtual Machine resources: %v", err)
		}

		if shouldTrackVirtualMachines {
			log.Info("Initializing virtual machine informers")
			virtualMachineInstanceInformer = k8swatch.NewInformerAdapter(
				gvrToAPIPath(virtualmachine.VirtualMachineInstance.GroupVersionResource()),
				k8sClient,
				func() runtime.Object { return &unstructured.Unstructured{} },
			)
			// Override the vmCrdHandlerFn to only handle when the resources become unavailable
			vmCrdHandlerFn = crdWatcherCallbackWrapper(k.context,
				resourcesUnavailable(),
				k.pubSub,
				"VirtualMachine resources have been removed. Connection will restart to force reconciliation with Central")
		}
		if err := vmWatcher.Watch(vmCrdHandlerFn); err != nil {
			log.Errorf("Failed to start watching the VirtualMachine CRDs: %v", err)
		}
	}

	// This call to clusterID.Get might block if a cluster ID is initially unavailable, which is okay.
	clusterID := k.clusterID.Get()

	// Create the pod informer using k8swatch adapter (not the typed SharedInformerFactory)
	// to avoid importing k8s.io/client-go/informers which pulls in all typed client packages.
	podInformer := k8swatch.NewInformerAdapter("/api/v1/pods", k8sClient, func() runtime.Object { return &corev1.Pod{} })
	podLister := v1Listers.NewPodLister(podInformer.GetIndexer())

	// Create the dispatcher registry, which provides dispatchers to all of the handlers.
	dispatchers := resources.NewDispatcherRegistry(
		clusterID,
		podLister,
		processfilter.Singleton(),
		k.configHandler,
		k.credentialsManager,
		k.traceWriter,
		k.storeProvider,
		k.client.Dynamic(),
	)

	namespaceInformer := k8swatch.NewInformerAdapter("/api/v1/namespaces", k8sClient, func() runtime.Object { return &corev1.Namespace{} })
	secretInformer := k8swatch.NewInformerAdapter("/api/v1/secrets", k8sClient, func() runtime.Object { return &corev1.Secret{} })
	saInformer := k8swatch.NewInformerAdapter("/api/v1/serviceaccounts", k8sClient, func() runtime.Object { return &corev1.ServiceAccount{} })

	roleInformer := k8swatch.NewInformerAdapter("/apis/rbac.authorization.k8s.io/v1/roles", k8sClient, func() runtime.Object { return &rbacv1.Role{} })
	clusterRoleInformer := k8swatch.NewInformerAdapter("/apis/rbac.authorization.k8s.io/v1/clusterroles", k8sClient, func() runtime.Object { return &rbacv1.ClusterRole{} })

	// The group that has no other object dependencies
	noDependencyWaitGroup := &concurrency.WaitGroup{}

	// we will single-thread event processing using this lock
	var eventLock sync.Mutex
	stopSignal := &k.stopSig

	// Informers that need to be synced initially
	handle(k.context, informerNamespaces, namespaceInformer, dispatchers.ForNamespaces(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerSecrets, secretInformer, dispatchers.ForSecrets(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerServiceAccounts, saInformer, dispatchers.ForServiceAccounts(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)

	// Roles need to be synced before role bindings because role bindings have a reference
	handle(k.context, informerRoles, roleInformer, dispatchers.ForRBAC(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerClusterRoles, clusterRoleInformer, dispatchers.ForRBAC(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)

	// OpenShift config and operator informers — using k8swatch adapters to avoid
	// importing typed OpenShift scheme packages (saves ~10 MB RSS on vanilla k8s).
	if isOpenShift {
		if resourceList, err := listenerUtils.ServerResourcesForGroup(k.client, osConfigGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osConfigGroupVersion, err)
		} else {
			if listenerUtils.ResourceExists(resourceList, osClusterOperatorsResourceName, osConfigGroupVersion) {
				log.Infof("Initializing %q informer", osClusterOperatorsResourceName)
				clusterOperatorInformer := k8swatch.NewInformerAdapter(
					gvrToAPIPath(clusterOperatorGVR),
					k8sClient,
					func() runtime.Object { return &unstructured.Unstructured{} },
				)
				handle(k.context, informerClusterOperators, clusterOperatorInformer, dispatchers.ForClusterOperators(), k.pubSubDispatcher, k.outputQueue, nil, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
			}

			if env.RegistryMirroringEnabled.BooleanSetting() {
				if listenerUtils.ResourceExists(resourceList, osImageDigestMirrorSetsResourceName, osConfigGroupVersion) {
					log.Infof("Initializing %q informer", osImageDigestMirrorSetsResourceName)
					imageDigestMirrorSetInformer := k8swatch.NewInformerAdapter(
						gvrToAPIPath(imageDigestMirrorSetGVR),
						k8sClient,
						func() runtime.Object { return &unstructured.Unstructured{} },
					)
					handle(k.context, informerImageDigestMirrorSets, imageDigestMirrorSetInformer, dispatchers.ForRegistryMirrors(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
				}

				if listenerUtils.ResourceExists(resourceList, osImageTagMirrorSetsResourceName, osConfigGroupVersion) {
					log.Infof("Initializing %q informer", osImageTagMirrorSetsResourceName)
					imageTagMirrorSetInformer := k8swatch.NewInformerAdapter(
						gvrToAPIPath(imageTagMirrorSetGVR),
						k8sClient,
						func() runtime.Object { return &unstructured.Unstructured{} },
					)
					handle(k.context, informerImageTagMirrorSets, imageTagMirrorSetInformer, dispatchers.ForRegistryMirrors(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
				}
			}
		}
	}

	if isOpenShift && env.RegistryMirroringEnabled.BooleanSetting() {
		if resourceList, err := listenerUtils.ServerResourcesForGroup(k.client, osOperatorAlphaGroupVersion); err != nil {
			log.Errorf("Checking API resources for group %q: %v", osOperatorAlphaGroupVersion, err)
		} else {
			if listenerUtils.ResourceExists(resourceList, osImageContentSourcePoliciesResourceName, osOperatorAlphaGroupVersion) {
				log.Infof("Initializing %q informer", osImageContentSourcePoliciesResourceName)
				imageContentSourceInformer := k8swatch.NewInformerAdapter(
					gvrToAPIPath(imageContentSourceGVR),
					k8sClient,
					func() runtime.Object { return &unstructured.Unstructured{} },
				)
				handle(k.context, informerImageContentSourcePolicies, imageContentSourceInformer, dispatchers.ForRegistryMirrors(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
			}
		}
	}

	if coAvailable {
		log.Info("Syncing compliance operator resources")
		// Handle results, rules, and scan setting bindings first
		handle(k.context, informerComplianceCheckResults, complianceResultInformer, dispatchers.ForComplianceOperatorResults(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		handle(k.context, informerComplianceRules, complianceRuleInformer, dispatchers.ForComplianceOperatorRules(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		handle(k.context, informerComplianceScanSettingBindings, complianceScanSettingBindingsInformer, dispatchers.ForComplianceOperatorScanSettingBindings(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		handle(k.context, informerComplianceScans, complianceScanInformer, dispatchers.ForComplianceOperatorScans(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		handle(k.context, informerComplianceSuites, complianceSuiteInformer, dispatchers.ForComplianceOperatorSuites(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		handle(k.context, informerComplianceRemediations, complianceRemediationInformer, dispatchers.ForComplianceOperatorRemediations(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)

		if customRulesAvailable {
			handle(k.context, informerComplianceCustomRules, complianceCustomRuleInformer, dispatchers.ForComplianceOperatorCustomRules(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
		}
	}

	if shouldTrackVirtualMachines {
		// We sync first the VirtualMachineInstances
		// This is because if both informers are racing in the sync, we could
		// send duplicate update events during sync
		log.Info("Syncing virtual machine instances")
		handle(k.context, informerVirtualMachineInstances, virtualMachineInstanceInformer, dispatchers.ForVirtualMachineInstances(), k.pubSubDispatcher, k.outputQueue, &syncingResources, noDependencyWaitGroup, stopSignal, &eventLock, informerTracker)
	}

	if !waitForWaitGroup(stopSignal, noDependencyWaitGroup) {
		return
	}
	log.Info("Successfully synced secrets, service accounts and roles")

	if shouldTrackVirtualMachines {
		// At this point the VirtualMachineInstances should be synced
		log.Info("Syncing virtual machines")
		virtualMachineInformer := k8swatch.NewInformerAdapter(
			gvrToAPIPath(virtualmachine.VirtualMachine.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		vmWaitGroup := &concurrency.WaitGroup{}
		handle(k.context, informerVirtualMachines, virtualMachineInformer, dispatchers.ForVirtualMachines(), k.pubSubDispatcher, k.outputQueue, &syncingResources, vmWaitGroup, stopSignal, &eventLock, informerTracker)
		if !waitForWaitGroup(stopSignal, vmWaitGroup) {
			return
		}
		log.Info("Successfully synced virtual machines")
	}

	// prePodWaitGroup
	prePodWaitGroup := &concurrency.WaitGroup{}

	roleBindingInformer := k8swatch.NewInformerAdapter("/apis/rbac.authorization.k8s.io/v1/rolebindings", k8sClient, func() runtime.Object { return &rbacv1.RoleBinding{} })
	clusterRoleBindingInformer := k8swatch.NewInformerAdapter("/apis/rbac.authorization.k8s.io/v1/clusterrolebindings", k8sClient, func() runtime.Object { return &rbacv1.ClusterRoleBinding{} })

	handle(k.context, informerRoleBindings, roleBindingInformer, dispatchers.ForRBAC(), k.pubSubDispatcher, k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerClusterRoleBindings, clusterRoleBindingInformer, dispatchers.ForRBAC(), k.pubSubDispatcher, k.outputQueue, &syncingResources, prePodWaitGroup, stopSignal, &eventLock, informerTracker)

	if !waitForWaitGroup(stopSignal, prePodWaitGroup) {
		return
	}

	log.Info("Successfully synced role bindings")

	// Wait for the pod informer to sync before processing other types.
	// This is required because the PodLister is used to populate the image ids of deployments.
	// However, do not ACTUALLY handle, pod events yet -- those need to wait for deployments to be
	// synced, since we need to enrich pods with the deployment ids, and for that we need the entire
	// hierarchy to be populated.
	// Start the pod adapter's watch goroutine so it can sync its cache.
	go podInformer.Run(stopSignal.Done())
	informerTracker.register(informerPodCache)
	if !cache.WaitForCacheSync(stopSignal.Done(), podInformer.HasSynced) {
		return
	}
	informerTracker.markSynced(informerPodCache)
	log.Info("Successfully synced k8s pod cache")

	preTopLevelDeploymentWaitGroup := &concurrency.WaitGroup{}

	// Non-deployment types.
	handle(k.context, informerNetworkPolicies, k8swatch.NewInformerAdapter("/apis/networking.k8s.io/v1/networkpolicies", k8sClient, func() runtime.Object { return &networkingv1.NetworkPolicy{} }), dispatchers.ForNetworkPolicies(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)

	// Nodes and Services use the minimal k8swatch watcher instead of client-go
	// informers. This saves 2 goroutines and the full object cache per resource,
	// and validates the minimal watcher pattern for broader adoption.
	handle(k.context, informerNodes,
		k8swatch.NewInformerAdapter("/api/v1/nodes", k8sClient, func() runtime.Object { return &corev1.Node{} }),
		dispatchers.ForNodes(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerServices,
		k8swatch.NewInformerAdapter("/api/v1/services", k8sClient, func() runtime.Object { return &corev1.Service{} }),
		dispatchers.ForServices(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)

	if isOpenShift {
		routeInformer := k8swatch.NewInformerAdapter(
			gvrToAPIPath(routeGVR),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		handle(k.context, informerRoutes, routeInformer, dispatchers.ForOpenshiftRoutes(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)
	}

	// Deployment subtypes (this ensures that the hierarchy maps are generated correctly)
	handle(k.context, informerJobs, k8swatch.NewInformerAdapter("/apis/batch/v1/jobs", k8sClient, func() runtime.Object { return &batchv1.Job{} }), dispatchers.ForJobs(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerReplicaSets, k8swatch.NewInformerAdapter("/apis/apps/v1/replicasets", k8sClient, func() runtime.Object { return &appsv1.ReplicaSet{} }), dispatchers.ForDeployments(kubernetesPkg.ReplicaSet), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerReplicationControllers, k8swatch.NewInformerAdapter("/api/v1/replicationcontrollers", k8sClient, func() runtime.Object { return &corev1.ReplicationController{} }), dispatchers.ForDeployments(kubernetesPkg.ReplicationController), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)

	// Compliance operator profiles are handled AFTER results, rules, and scan setting bindings have been synced
	if coAvailable {
		complianceProfileInformer := k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.Profile.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		// Create a fake lister that satisfies the interface but doesn't cache objects.
		// Compliance profiles are only accessed via events, not direct lookups.
		profileLister = &noOpGenericLister{}
		handle(k.context, informerComplianceProfiles, complianceProfileInformer, dispatchers.ForComplianceOperatorProfiles(), k.pubSubDispatcher, k.outputQueue, &syncingResources, preTopLevelDeploymentWaitGroup, stopSignal, &eventLock, informerTracker)
	}

	if !waitForWaitGroup(stopSignal, preTopLevelDeploymentWaitGroup) {
		return
	}

	log.Info("Successfully synced network policies, nodes, services, jobs, replica sets, and replication controllers")

	wg := &concurrency.WaitGroup{}

	// Deployment types.
	handle(k.context, informerDaemonSets, k8swatch.NewInformerAdapter("/apis/apps/v1/daemonsets", k8sClient, func() runtime.Object { return &appsv1.DaemonSet{} }), dispatchers.ForDeployments(kubernetesPkg.DaemonSet), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerDeployments, k8swatch.NewInformerAdapter("/apis/apps/v1/deployments", k8sClient, func() runtime.Object { return &appsv1.Deployment{} }), dispatchers.ForDeployments(kubernetesPkg.Deployment), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)
	handle(k.context, informerStatefulSets, k8swatch.NewInformerAdapter("/apis/apps/v1/statefulsets", k8sClient, func() runtime.Object { return &appsv1.StatefulSet{} }), dispatchers.ForDeployments(kubernetesPkg.StatefulSet), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)

	// k8swatch adapter uses JSON, so it handles both v1 and v1beta1 CronJobs
	handle(k.context, informerCronJobs, k8swatch.NewInformerAdapter("/apis/batch/v1/cronjobs", k8sClient, func() runtime.Object { return &batchv1.CronJob{} }), dispatchers.ForDeployments(kubernetesPkg.CronJob), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)
	if isOpenShift {
		deploymentConfigInformer := k8swatch.NewInformerAdapter(
			gvrToAPIPath(deploymentConfigGVR),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		handle(k.context, informerDeploymentConfigs, deploymentConfigInformer, dispatchers.ForDeployments(kubernetesPkg.DeploymentConfig), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)
	}

	// Compliance operator tailored profiles may depend on non-tailored profiles, so we need to start the informer after those were synced
	if coAvailable {
		complianceTailoredProfileInformer := k8swatch.NewInformerAdapter(
			gvrToAPIPath(complianceoperator.TailoredProfile.GroupVersionResource()),
			k8sClient,
			func() runtime.Object { return &unstructured.Unstructured{} },
		)
		handle(k.context, informerComplianceTailoredProfiles, complianceTailoredProfileInformer, dispatchers.ForComplianceOperatorTailoredProfiles(profileLister), k.pubSubDispatcher, k.outputQueue, &syncingResources, wg, stopSignal, &eventLock, informerTracker)
	}

	if !waitForWaitGroup(stopSignal, wg) {
		return
	}

	log.Info("Successfully synced daemonsets, deployments, stateful sets and cronjobs")

	// Finally, process pod events. The pod informer is already running (started above for cache sync).
	podWaitGroup := &concurrency.WaitGroup{}
	handle(k.context, informerPods, podInformer, dispatchers.ForDeployments(kubernetesPkg.Pod), k.pubSubDispatcher, k.outputQueue, &syncingResources, podWaitGroup, stopSignal, &eventLock, informerTracker)
	if !waitForWaitGroup(stopSignal, podWaitGroup) {
		return
	}

	log.Info("Successfully synced pods")

	// Set the flag that all objects present at start up have been consumed.
	syncingResources.Set(false)

	syncedEvent := &component.ResourceEvent{
		ForwardMessages: []*central.SensorEvent{
			{
				Resource: &central.SensorEvent_Synced{
					Synced: &central.SensorEvent_ResourcesSynced{},
				},
			},
		},
		Context: k.context,
	}

	if features.SensorInternalPubSub.Enabled() {
		if err := k.pubSubDispatcher.Publish(syncedEvent); err != nil {
			log.Errorf("unable to publish synced event: topic=%q, lane=%q: %v",
				syncedEvent.Topic().String(),
				syncedEvent.Lane().String(),
				err)
			return
		}
	} else {
		k.outputQueue.Send(syncedEvent)
	}
	utils.Should(k.pubSub.Publish(&internalmessage.SensorInternalMessage{
		Kind:     internalmessage.SensorMessageResourceSyncFinished,
		Text:     "Finished the k8s resource sync",
		Validity: k.context,
	}))
}

// Helper function that creates and adds a handler to an informer.
// The name parameter identifies the informer for sync tracking.
// The tracker parameter may be nil when the watchdog feature is disabled.
// stripCacheTransform removes fields from cached objects that sensor never reads.
// This reduces informer cache memory by stripping bulky metadata like
// last-applied-configuration annotations and managedFields.
func stripCacheTransform(obj interface{}) (interface{}, error) {
	if accessor, ok := obj.(v1.ObjectMetaAccessor); ok {
		meta := accessor.GetObjectMeta()
		// last-applied-configuration is a duplicate of the entire spec
		// stored as a JSON annotation — often 50%+ of the object size.
		annotations := meta.GetAnnotations()
		if _, exists := annotations["kubectl.kubernetes.io/last-applied-configuration"]; exists {
			filtered := make(map[string]string, len(annotations)-1)
			for k, v := range annotations {
				if k != "kubectl.kubernetes.io/last-applied-configuration" {
					filtered[k] = v
				}
			}
			meta.SetAnnotations(filtered)
		}
		// managedFields tracks field ownership for server-side apply.
		// Sensor never reads them.
		meta.SetManagedFields(nil)
	}
	return obj, nil
}

func handle(
	ctx context.Context,
	name string,
	informer cache.SharedIndexInformer,
	dispatcher resources.Dispatcher,
	pubSubDispatcher pubSubPublisher,
	resolver component.Resolver,
	syncingResources *concurrency.Flag,
	wg *concurrency.WaitGroup,
	stopSignal *concurrency.Signal,
	eventLock *sync.Mutex,
	tracker *informerSyncTracker,
) {
	// If this is a k8swatch adapter (not a real informer), start its watch goroutine.
	// Real informers are started by SharedInformerFactory.Start(); adapters manage themselves.
	if adapter, ok := informer.(*k8swatch.InformerAdapter); ok {
		go adapter.Run(stopSignal.Done())
	} else {
		// Strip unnecessary fields before caching to reduce memory.
		if err := informer.SetTransform(stripCacheTransform); err != nil {
			log.Warnf("Failed to set transform for informer %s: %v", name, err)
		}
	}
	tracker.register(name)
	utils.Should(func() error {
		if features.SensorInternalPubSub.Enabled() && pubSubDispatcher == nil {
			return errors.Errorf("informer `handle` was called with a `nil` PubSubDispatcher when %q is enabled", features.SensorInternalPubSub.EnvVar())
		}
		return nil
	}())
	handlerImpl := &resourceEventHandlerImpl{
		context:          ctx,
		eventLock:        eventLock,
		dispatcher:       dispatcher,
		syncingResources: syncingResources,

		resolver:         resolver,
		pubSubDispatcher: pubSubDispatcher,

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
			log.Warnf("Informer %q: cache sync wait aborted", name)
			return
		}
		tracker.markSynced(name)
		initialObjects := informer.GetIndexer().List()
		doneChannel := handlerImpl.PopulateInitialObjects(initialObjects)
		waitStarted := time.Now()
		warnTicker := time.NewTicker(15 * time.Second)
		defer warnTicker.Stop()
		for {
			select {
			case <-stopSignal.Done():
				log.Infof("Informer %q: initial object population wait interrupted after %s", name, time.Since(waitStarted).Truncate(time.Millisecond))
				return
			case <-doneChannel:
				duration := time.Since(waitStarted)
				sensorMetrics.ObserveInformerInitialObjectPopulationDuration(name, duration)
				log.Debugf("Informer %q: initial object population completed in %s", name, duration.Truncate(time.Millisecond))
				return
			case <-warnTicker.C:
				missingCount, totalCount := handlerImpl.initialSyncDebugState()
				log.Infof(
					"Informer %q: still waiting for initial object population after %s (missing=%d total=%d)",
					name,
					time.Since(waitStarted).Truncate(time.Millisecond),
					missingCount,
					totalCount,
				)
			}
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
