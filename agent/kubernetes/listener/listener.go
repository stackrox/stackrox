package listener

import (
	"time"

	pkgV1 "bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

const (
	resyncPeriod = 10 * time.Minute
)

var (
	logger = logging.New("listener")
)

type kubernetesListener struct {
	client *kubernetes.Clientset

	metaC   chan resourceWrap
	eventsC chan *pkgV1.DeploymentEvent

	resourcesWL []resourceWatchLister
}

// New returns a new kubernetes listener.
func New() listeners.Listener {
	k := &kubernetesListener{
		metaC:   make(chan resourceWrap, 10),
		eventsC: make(chan *pkgV1.DeploymentEvent, 10),
	}
	k.initialize()
	return k
}

func (k *kubernetesListener) initialize() {
	k.setupClient()
	k.createResourceWatchers()
}

func (k *kubernetesListener) Start() {
	for _, wl := range k.resourcesWL {
		go wl.watch()
	}

	go k.listenForDeployments()
}

func (k *kubernetesListener) createResourceWatchers() {
	k.resourcesWL = []resourceWatchLister{
		newReplicaSetWatchLister(k.client.ExtensionsV1beta1().RESTClient(), k.metaC),
		newDaemonSetWatchLister(k.client.ExtensionsV1beta1().RESTClient(), k.metaC),
		newReplicationControllerWatchLister(k.client.CoreV1().RESTClient(), k.metaC),
		newDeploymentWatcher(k.client.ExtensionsV1beta1().RESTClient(), k.metaC),
		newStatefulSetWatchLister(k.client.AppsV1beta1().RESTClient(), k.metaC),
	}
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
	for _, wl := range k.resourcesWL {
		wl.stop()
	}
}

func (k *kubernetesListener) listenForDeployments() {
	for metaObj := range k.metaC {
		deployment := metaObj.asDeploymentEvent()

		k.eventsC <- deployment
	}
}

func (k *kubernetesListener) Events() <-chan *pkgV1.DeploymentEvent {
	return k.eventsC
}

type watchLister struct {
	client     rest.Interface
	store      cache.Store
	controller cache.Controller
	stopC      chan (struct{})
}

func newWatchLister(client rest.Interface) watchLister {
	return watchLister{
		client: client,
		stopC:  make(chan struct{}),
	}
}

func (wl *watchLister) watch(object string, objectType runtime.Object, changedFunc func(interface{}, pkgV1.ResourceAction)) {
	watchlist := cache.NewListWatchFromClient(wl.client, object, api.NamespaceAll, fields.Everything())

	wl.store, wl.controller = cache.NewInformer(
		watchlist,
		objectType,
		resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				changedFunc(obj, pkgV1.ResourceAction_CREATE_RESOURCE)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				changedFunc(newObj, pkgV1.ResourceAction_UPDATE_RESOURCE)
			},
			DeleteFunc: func(obj interface{}) {
				changedFunc(obj, pkgV1.ResourceAction_REMOVE_RESOURCE)
			},
		},
	)

	wl.controller.Run(wl.stopC)
}

func (wl *watchLister) stop() {
	wl.stopC <- struct{}{}
}
