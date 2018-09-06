package resources

import (
	"reflect"
	"strings"
	"time"

	ocappsv1 "github.com/openshift/api/apps/v1"
	pkgV1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/kubernetes"
	"github.com/stackrox/rox/pkg/listeners"
	"github.com/stackrox/rox/sensor/kubernetes/listener/watchlister"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

// ResourceWatchLister is the generic interface for watching a resource
type ResourceWatchLister interface {
	Watch(serviceWatchLister *ServiceWatchLister)
	Stop()
	Initialize()
	ListObjects() (objs []interface{}, deployments []*pkgV1.Deployment)
	ResourceType() string
}

// A reflectionWatchLister extracts the ObjectMetadata using reflection.
type reflectionWatchLister struct {
	watchlister.WatchLister
	rt                     string
	objectType             runtime.Object
	metaFieldIndex         []int
	initialObjectsConsumed bool

	podLister          podLister
	serviceWatchLister *ServiceWatchLister

	eventC chan<- *listeners.EventWrap
}

func newReflectionWatcherFromClient(client rest.Interface, resourceType string, objectType runtime.Object, eventC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) *reflectionWatchLister {
	return newReflectionWatcher(watchlister.NewWatchLister(client, resyncPeriod), resourceType, objectType, eventC, lister)
}

func newReflectionWatcher(watchLister watchlister.WatchLister, resourceType string, objectType runtime.Object, eventC chan<- *listeners.EventWrap, lister podLister) *reflectionWatchLister {
	ty := reflect.Indirect(reflect.ValueOf(objectType)).Type()
	metaField, ok := ty.FieldByName("ObjectMeta")
	if !ok || metaField.Type != reflect.TypeOf(metav1.ObjectMeta{}) {
		logger.Errorf("Type %s does not have an ObjectMeta field", ty.Name())
		return nil
	}

	return &reflectionWatchLister{
		WatchLister:    watchLister,
		rt:             resourceType,
		objectType:     objectType,
		metaFieldIndex: metaField.Index,
		podLister:      lister,
		eventC:         eventC,
	}
}

func (wl *reflectionWatchLister) Watch(serviceWatchLister *ServiceWatchLister) {
	// We use the lowercase'd version of the resource type plus a plural "s" as the type of objects to watch.
	wl.serviceWatchLister = serviceWatchLister
	wl.SetupWatch(strings.ToLower(wl.rt)+"s", wl.objectType, wl.resourceChanged)
	go wl.WatchLister.StartWatch()
	go wl.Initialize()
}

// ResourceType returns the type being watched
func (wl *reflectionWatchLister) ResourceType() string {
	return wl.rt
}

// Stop stops the reflection watch lister
func (wl *reflectionWatchLister) Stop() {
	wl.WatchLister.Stop()
}

func (wl *reflectionWatchLister) resourceChanged(obj interface{}, action pkgV1.ResourceAction) {
	// If the action is create, then it came from watchlister AddFunc.
	// If we are listing the initial objects, then we treat them as updates so enforcement isn't done
	if !wl.initialObjectsConsumed && action == pkgV1.ResourceAction_CREATE_RESOURCE {
		action = pkgV1.ResourceAction_UPDATE_RESOURCE
	}

	if d := newDeploymentEventFromResource(obj, action, wl.metaFieldIndex, wl.ResourceType(), wl.podLister); d != nil {
		wl.serviceWatchLister.updatePortExposureFromStore(d)

		wl.eventC <- &listeners.EventWrap{
			SensorEvent: &pkgV1.SensorEvent{
				Id:     d.GetId(),
				Action: action,
				Resource: &pkgV1.SensorEvent_Deployment{
					Deployment: d,
				},
			},
			OriginalSpec: obj,
		}
	}
}

// Initialize periodically checks whether the watchLister has made an initial sync to retrieve preexisting objects.
// Subsequent objects processed are assumed to be new, i.e. a CREATE_RESOURCE action.
func (wl *reflectionWatchLister) Initialize() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if wl.WatchLister.Controller != nil && wl.WatchLister.Controller.HasSynced() {
			wl.initialObjectsConsumed = true
			return
		}
	}
}

// ListObjects gets all of the current deployments from existing resources
func (wl *reflectionWatchLister) ListObjects() (objs []interface{}, deploymentEvents []*pkgV1.Deployment) {
	for _, obj := range wl.Store.List() {
		if d := newDeploymentEventFromResource(obj, pkgV1.ResourceAction_UPDATE_RESOURCE, wl.metaFieldIndex, wl.ResourceType(), wl.podLister); d != nil {
			objs = append(objs, obj)
			deploymentEvents = append(deploymentEvents, d)
		}
	}
	return
}

// NewReplicaSetWatchLister initializes the replica set watch
func NewReplicaSetWatchLister(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.ReplicaSet, &v1beta1.ReplicaSet{}, eventsC, lister, resyncPeriod)
}

// NewDaemonSetWatchLister initializes the daemon set watch
func NewDaemonSetWatchLister(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.DaemonSet, &v1beta1.DaemonSet{}, eventsC, lister, resyncPeriod)
}

// NewReplicationControllerWatchLister initializes replication controller watch
func NewReplicationControllerWatchLister(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.ReplicationController, &v1.ReplicationController{}, eventsC, lister, resyncPeriod)
}

// NewPodWatcher initializes pod watch
func NewPodWatcher(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.Pod, &v1.Pod{}, eventsC, lister, resyncPeriod)
}

// NewDeploymentWatcher initializes deployment watch
func NewDeploymentWatcher(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.Deployment, &v1beta1.Deployment{}, eventsC, lister, resyncPeriod)
}

// NewStatefulSetWatchLister initializes stateful set watch
func NewStatefulSetWatchLister(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.StatefulSet, &appsv1beta1.StatefulSet{}, eventsC, lister, resyncPeriod)
}

// NewDeploymentConfigWatcher initializes deployment config watch that is specific for OpenShift
func NewDeploymentConfigWatcher(client rest.Interface, eventsC chan<- *listeners.EventWrap, lister podLister, resyncPeriod time.Duration) ResourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.DeploymentConfig, &ocappsv1.DeploymentConfig{}, eventsC, lister, resyncPeriod)
}
