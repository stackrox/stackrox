package listener

import (
	"time"

	openshift "github.com/openshift/client-go/apps/clientset/versioned"
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/listener/namespace"
	"github.com/stackrox/rox/sensor/kubernetes/listener/networkpolicy"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources"
	"github.com/stackrox/rox/sensor/kubernetes/listener/secret"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	resyncPeriod = 10 * time.Minute
)

var (
	logger = logging.LoggerForModule()
)

type clientSet struct {
	k8s       *kubernetes.Clientset
	openshift *openshift.Clientset
}

type kubernetesListener struct {
	clients *clientSet
	eventsC chan *listeners.EventWrap

	podWL       *resources.PodWatchLister
	resourcesWL []resources.ResourceWatchLister
	serviceWL   *resources.ServiceWatchLister

	networkPolicyWL *networkpolicy.WatchLister
	namespaceWL     *namespace.WatchLister
	secretWL        *secret.WatchLister
}

// New returns a new kubernetes listener.
func New() listeners.Listener {
	k := &kubernetesListener{
		eventsC: make(chan *listeners.EventWrap, 10),
	}
	k.initialize()
	return k
}

func (k *kubernetesListener) initialize() {
	k.setupClient()
	k.createResourceWatchers()
	k.createNetworkPolicyWatcher()
	k.createNamespaceWatcher()
	k.createSecretWatcher()
}

func (k *kubernetesListener) Start() {
	go k.podWL.Watch()
	k.podWL.BlockUntilSynced()

	for _, wl := range k.resourcesWL {
		go wl.Watch(k.serviceWL)
	}

	go k.serviceWL.StartWatch()
	go k.networkPolicyWL.StartWatch()
	go k.secretWL.StartWatch()
}

func (k *kubernetesListener) createResourceWatchers() {
	k.podWL = resources.NewPodWatchLister(k.clients.k8s.CoreV1().RESTClient(), resyncPeriod)

	k.resourcesWL = []resources.ResourceWatchLister{
		resources.NewReplicaSetWatchLister(k.clients.k8s.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL, resyncPeriod),
		resources.NewDaemonSetWatchLister(k.clients.k8s.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL, resyncPeriod),
		resources.NewReplicationControllerWatchLister(k.clients.k8s.CoreV1().RESTClient(), k.eventsC, k.podWL, resyncPeriod),
		resources.NewDeploymentWatcher(k.clients.k8s.ExtensionsV1beta1().RESTClient(), k.eventsC, k.podWL, resyncPeriod),
		resources.NewStatefulSetWatchLister(k.clients.k8s.AppsV1beta1().RESTClient(), k.eventsC, k.podWL, resyncPeriod),
	}

	if env.OpenshiftAPI.Setting() == "true" {
		k.resourcesWL = append(k.resourcesWL, resources.NewDeploymentConfigWatcher(k.clients.openshift.AppsV1().RESTClient(), k.eventsC, k.podWL, resyncPeriod))
	}

	var deploymentGetters []func() (objs []interface{}, deploymentEvents []*pkgV1.Deployment)
	for _, wl := range k.resourcesWL {
		deploymentGetters = append(deploymentGetters, wl.ListObjects)
	}

	k.serviceWL = resources.NewServiceWatchLister(k.clients.k8s.CoreV1().RESTClient(), k.eventsC, resyncPeriod, deploymentGetters...)
}

func (k *kubernetesListener) createNetworkPolicyWatcher() {
	k.networkPolicyWL = networkpolicy.NewWatchLister(k.clients.k8s.NetworkingV1().RESTClient(), k.eventsC, resyncPeriod)
}

func (k *kubernetesListener) createNamespaceWatcher() {
	k.namespaceWL = namespace.NewWatchLister(k.clients.k8s.CoreV1().RESTClient(), k.eventsC, resyncPeriod)
}

func (k *kubernetesListener) createSecretWatcher() {
	k.secretWL = secret.NewWatchLister(k.clients.k8s.CoreV1().RESTClient(), k.eventsC, resyncPeriod)
}

func (k *kubernetesListener) setupClient() {
	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatalf("Unable to get cluster config: %s", err)
	}

	k8s, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatalf("Unable to get k8s client: %s", err)
	}

	oc, err := openshift.NewForConfig(config)
	if err != nil {
		logger.Warnf("Could not generate openshift client: %s", err)
	}

	k.clients = &clientSet{
		k8s:       k8s,
		openshift: oc,
	}
}

func (k *kubernetesListener) Stop() {
	k.podWL.Stop()

	for _, wl := range k.resourcesWL {
		wl.Stop()
	}

	k.serviceWL.Stop()
}

func (k *kubernetesListener) Events() <-chan *listeners.EventWrap {
	return k.eventsC
}
