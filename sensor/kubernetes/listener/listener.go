package listener

import (
	"time"

	pkgV1 "bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod = 10 * time.Minute
)

var (
	logger = logging.LoggerForModule()
)

type kubernetesListener struct {
	client *kubernetes.Clientset

	eventsC chan *listeners.DeploymentEventWrap

	podWL       *podWatchLister
	resourcesWL []resourceWatchLister
	serviceWL   *serviceWatchLister
}

// New returns a new kubernetes listener.
func New() listeners.Listener {
	k := &kubernetesListener{
		eventsC: make(chan *listeners.DeploymentEventWrap, 10),
	}
	k.initialize()
	return k
}

func (k *kubernetesListener) initialize() {
	k.setupClient()
	k.createResourceWatchers()
}

func (k *kubernetesListener) Start() {
	go k.podWL.watch()
	k.podWL.blockUntilSynced()

	for _, wl := range k.resourcesWL {
		go wl.watch(k.serviceWL)
	}

	go k.serviceWL.startWatch()
}

func (k *kubernetesListener) createResourceWatchers() {
	k.podWL = newPodWatchLister(k.client.CoreV1().RESTClient())

	k.resourcesWL = []resourceWatchLister{
		newReplicaSetWatchLister(k.client.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL),
		newDaemonSetWatchLister(k.client.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL),
		newReplicationControllerWatchLister(k.client.CoreV1().RESTClient(), k.eventsC, k.podWL),
		newDeploymentWatcher(k.client.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL),
		newStatefulSetWatchLister(k.client.AppsV1beta1().RESTClient(), k.eventsC, k.podWL),
	}

	var deploymentGetters []func() (objs []interface{}, deploymentEvents []*pkgV1.DeploymentEvent)
	for _, wl := range k.resourcesWL {
		deploymentGetters = append(deploymentGetters, wl.listObjects)
	}

	k.serviceWL = newServiceWatchLister(k.client.CoreV1().RESTClient(), k.eventsC, deploymentGetters...)
}

func (k *kubernetesListener) setupClient() {
	c, err := getClient()
	if err != nil {
		logger.Fatalf("Unable to get kubernetes client")
	}

	k.client = c
}

func getClient() (client *kubernetes.Clientset, err error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return
	}

	return kubernetes.NewForConfig(config)
}

func (k *kubernetesListener) Stop() {
	k.podWL.stop()

	for _, wl := range k.resourcesWL {
		wl.stop()
	}

	k.serviceWL.stop()
}

func (k *kubernetesListener) Events() <-chan *listeners.DeploymentEventWrap {
	return k.eventsC
}

type watchLister struct {
	client     rest.Interface
	store      cache.Store
	controller cache.Controller
	stopC      chan struct{}
}

func newWatchLister(client rest.Interface) watchLister {
	return watchLister{
		client: client,
		stopC:  make(chan struct{}),
	}
}

func (wl *watchLister) setupWatch(object string, objectType runtime.Object, changedFunc func(interface{}, pkgV1.ResourceAction)) {
	watchlist := cache.NewListWatchFromClient(wl.client, object, v1.NamespaceAll, fields.Everything())

	wl.store, wl.controller = cache.NewInformer(
		watchlist,
		objectType,
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				// Once the initial objects are listed, the resource action changes to CREATE.
				changedFunc(obj, pkgV1.ResourceAction_PREEXISTING_RESOURCE)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				changedFunc(newObj, pkgV1.ResourceAction_UPDATE_RESOURCE)
			},
			DeleteFunc: func(obj interface{}) {
				changedFunc(obj, pkgV1.ResourceAction_REMOVE_RESOURCE)
			},
		},
	)
}

func (wl *watchLister) startWatch() {
	wl.controller.Run(wl.stopC)
}

func (wl *watchLister) stop() {
	wl.stopC <- struct{}{}
}
