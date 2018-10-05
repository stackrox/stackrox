package listener

import (
	"reflect"
	"time"

	openshift "github.com/openshift/client-go/apps/clientset/versioned"
	"github.com/openshift/client-go/apps/informers/externalversions"
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/common/clusterentities"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"k8s.io/client-go/informers"
	kubernetesClient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod = 10 * time.Minute
)

var (
	logger = logging.LoggerForModule()
)

type informerFactoryInterface interface {
	Start(stopCh <-chan struct{})
	WaitForCacheSync(stopCh <-chan struct{}) map[reflect.Type]bool
}

type informerGetter interface {
	Informer() cache.SharedIndexInformer
}

type clientSet struct {
	k8s       *kubernetesClient.Clientset
	openshift *openshift.Clientset
}

type kubernetesListener struct {
	clients *clientSet
	eventsC chan *listeners.EventWrap

	resourceEventsC chan resourceEvent

	resourceEventDispatcher resources.Dispatcher

	stopSig                concurrency.Signal
	initialObjectsConsumed concurrency.Flag
}

// New returns a new kubernetes listener.
func New() listeners.Listener {
	k := &kubernetesListener{
		clients:         createClient(),
		eventsC:         make(chan *listeners.EventWrap, 10),
		resourceEventsC: make(chan resourceEvent),
		stopSig:         concurrency.NewSignal(),
	}
	return k
}

type resourceEvent struct {
	obj            interface{}
	action         pkgV1.ResourceAction
	deploymentType string
}

func (k *kubernetesListener) sendResourceEvent(obj interface{}, action pkgV1.ResourceAction, deploymentType string) {
	rev := resourceEvent{
		obj:            obj,
		action:         action,
		deploymentType: deploymentType,
	}
	// If the action is create, then it came from watchlister AddFunc.
	// If we are listing the initial objects, then we treat them as updates so enforcement isn't done
	if deploymentType != "" && action == pkgV1.ResourceAction_CREATE_RESOURCE && !k.initialObjectsConsumed.Get() {
		rev.action = pkgV1.ResourceAction_UPDATE_RESOURCE
	}

	select {
	case k.resourceEventsC <- rev:
	case <-k.stopSig.Done():
	}
}

type resourceEventHandler struct {
	listener       *kubernetesListener
	deploymentType string
}

func (h resourceEventHandler) OnAdd(obj interface{}) {
	h.listener.sendResourceEvent(obj, pkgV1.ResourceAction_CREATE_RESOURCE, h.deploymentType)
}

func (h resourceEventHandler) OnUpdate(oldObj, newObj interface{}) {
	h.listener.sendResourceEvent(newObj, pkgV1.ResourceAction_UPDATE_RESOURCE, h.deploymentType)
}

func (h resourceEventHandler) OnDelete(obj interface{}) {
	h.listener.sendResourceEvent(obj, pkgV1.ResourceAction_REMOVE_RESOURCE, h.deploymentType)
}

func (k *kubernetesListener) Start() {
	k8sFactory := informers.NewSharedInformerFactory(k.clients.k8s, resyncPeriod)
	podInformer := k8sFactory.Core().V1().Pods()
	deploymentResources := map[string]informerGetter{
		kubernetes.Pod:                   podInformer,
		kubernetes.ReplicationController: k8sFactory.Core().V1().ReplicationControllers(),
		kubernetes.DaemonSet:             k8sFactory.Extensions().V1beta1().DaemonSets(),
		kubernetes.Deployment:            k8sFactory.Extensions().V1beta1().Deployments(),
		kubernetes.ReplicaSet:            k8sFactory.Extensions().V1beta1().ReplicaSets(),
		kubernetes.StatefulSet:           k8sFactory.Apps().V1beta1().StatefulSets(),
	}
	watchResources := []informerGetter{
		k8sFactory.Core().V1().Secrets(),
		k8sFactory.Core().V1().Services(),
		k8sFactory.Core().V1().Namespaces(),
		k8sFactory.Networking().V1().NetworkPolicies(),
	}

	factories := []informerFactoryInterface{k8sFactory}

	if env.OpenshiftAPI.Setting() == "true" {
		factory := externalversions.NewSharedInformerFactory(k.clients.openshift, resyncPeriod)
		deploymentResources[kubernetes.DeploymentConfig] = factory.Apps().V1().DeploymentConfigs()
		factories = append(factories, factory)
	}

	k.registerDeploymentEventHandlers(deploymentResources)
	k.registerEventHandlers(watchResources)

	k.resourceEventDispatcher = resources.NewDispatcher(podInformer.Lister(), clusterentities.StoreInstance())

	go k.processResourceEvents()

	go podInformer.Informer().Run(k.stopSig.Done())
	// Wait for the pod informer to have synced
	cache.WaitForCacheSync(k.stopSig.Done(), podInformer.Informer().HasSynced)

	for _, informer := range factories {
		informer.Start(k.stopSig.Done())
	}
	for _, informer := range factories {
		informer.WaitForCacheSync(k.stopSig.Done())
	}
	k.initialObjectsConsumed.Set(true)
}

func (k *kubernetesListener) registerEventHandlers(informerGetters []informerGetter) {
	handler := resourceEventHandler{
		listener: k,
	}
	for _, ig := range informerGetters {
		ig.Informer().AddEventHandler(handler)
	}
}

func (k *kubernetesListener) registerDeploymentEventHandlers(informerGetters map[string]informerGetter) {
	for deploymentType, ig := range informerGetters {
		handler := resourceEventHandler{
			listener:       k,
			deploymentType: deploymentType,
		}
		ig.Informer().AddEventHandler(handler)
	}
}

func createClient() *clientSet {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("Unable to get cluster config: %s", err)
	}

	k8s, err := kubernetesClient.NewForConfig(config)
	if err != nil {
		logger.Fatalf("Unable to get k8s client: %s", err)
	}

	oc, err := openshift.NewForConfig(config)
	if err != nil {
		logger.Warnf("Could not generate openshift client: %s", err)
	}

	return &clientSet{
		k8s:       k8s,
		openshift: oc,
	}
}

func (k *kubernetesListener) Stop() {
	k.stopSig.Signal()
}

func (k *kubernetesListener) Events() <-chan *listeners.EventWrap {
	return k.eventsC
}

func (k *kubernetesListener) processResourceEvents() {
	for {
		select {
		case resourceEv, ok := <-k.resourceEventsC:
			if !ok {
				return
			}
			evWraps := k.resourceEventDispatcher.ProcessEvent(resourceEv.obj, resourceEv.action, resourceEv.deploymentType)
			k.sendEvents(evWraps...)
		case <-k.stopSig.Done():
			return
		}
	}
}

func (k *kubernetesListener) sendEvents(evWraps ...*listeners.EventWrap) {
	for _, evWrap := range evWraps {
		select {
		case k.eventsC <- evWrap:
		case <-k.stopSig.Done():
			return
		}
	}
}
