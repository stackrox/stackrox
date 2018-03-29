package listener

import (
	"reflect"
	"strings"
	"time"

	pkgV1 "bitbucket.org/stack-rox/apollo/generated/api/v1"
	"bitbucket.org/stack-rox/apollo/pkg/kubernetes"
	"bitbucket.org/stack-rox/apollo/pkg/listeners"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

type resourceWatchLister interface {
	watch(serviceWatchLister *serviceWatchLister)
	stop()
	initialize()
	listObjects() (objs []interface{}, deploymentEvents []*pkgV1.DeploymentEvent)
	resourceType() string
}

// A reflectionWatchLister extracts the ObjectMetadata using reflection.
type reflectionWatchLister struct {
	watchLister
	rt                     string
	objectType             runtime.Object
	metaFieldIndex         []int
	initialObjectsConsumed bool

	podLister          podLister
	serviceWatchLister *serviceWatchLister

	eventC chan<- *listeners.DeploymentEventWrap
}

func newReflectionWatcherFromClient(client rest.Interface, resourceType string, objectType runtime.Object, eventC chan<- *listeners.DeploymentEventWrap, lister podLister) *reflectionWatchLister {
	return newReflectionWatcher(newWatchLister(client), resourceType, objectType, eventC, lister)
}

func newReflectionWatcher(watchLister watchLister, resourceType string, objectType runtime.Object, eventC chan<- *listeners.DeploymentEventWrap, lister podLister) *reflectionWatchLister {
	ty := reflect.Indirect(reflect.ValueOf(objectType)).Type()
	metaField, ok := ty.FieldByName("ObjectMeta")
	if !ok || metaField.Type != reflect.TypeOf(metav1.ObjectMeta{}) {
		logger.Errorf("Type %s does not have an ObjectMeta field", ty.Name())
		return nil
	}

	return &reflectionWatchLister{
		watchLister:    watchLister,
		rt:             resourceType,
		objectType:     objectType,
		metaFieldIndex: metaField.Index,
		podLister:      lister,
		eventC:         eventC,
	}
}

func (wl *reflectionWatchLister) watch(serviceWatchLister *serviceWatchLister) {
	// We use the lowercase'd version of the resource type plus a plural "s" as the type of objects to watch.
	wl.serviceWatchLister = serviceWatchLister
	wl.setupWatch(strings.ToLower(wl.rt)+"s", wl.objectType, wl.resourceChanged)
	go wl.watchLister.startWatch()
	go wl.initialize()
}

func (wl *reflectionWatchLister) resourceType() string {
	return wl.rt
}

func (wl *reflectionWatchLister) stop() {
	wl.watchLister.stop()
}

func (wl *reflectionWatchLister) resourceChanged(obj interface{}, action pkgV1.ResourceAction) {
	if wl.initialObjectsConsumed && action == pkgV1.ResourceAction_PREEXISTING_RESOURCE {
		action = pkgV1.ResourceAction_CREATE_RESOURCE
	}

	if d := newDeploymentEventFromResource(obj, action, wl.metaFieldIndex, wl.resourceType(), wl.podLister); d != nil {
		wl.serviceWatchLister.updatePortExposureFromStore(d)

		wl.eventC <- &listeners.DeploymentEventWrap{
			DeploymentEvent: d,
			OriginalSpec:    obj,
		}
	}
}

// initialize periodically checks whether the watchLister has made an initial sync to retrieve preexisting objects.
// Subsequent objects processed are assumed to be new, i.e. a CREATE_RESOURCE action.
func (wl *reflectionWatchLister) initialize() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if wl.watchLister.controller != nil && wl.watchLister.controller.HasSynced() {
			wl.initialObjectsConsumed = true
			return
		}
	}
}

func (wl *reflectionWatchLister) listObjects() (objs []interface{}, deploymentEvents []*pkgV1.DeploymentEvent) {
	for _, obj := range wl.store.List() {
		if d := newDeploymentEventFromResource(obj, pkgV1.ResourceAction_UPDATE_RESOURCE, wl.metaFieldIndex, wl.resourceType(), wl.podLister); d != nil {
			objs = append(objs, obj)
			deploymentEvents = append(deploymentEvents, d)
		}
	}
	return
}

// Factory methods for the types of resources we support.

func newReplicaSetWatchLister(client rest.Interface, eventsC chan<- *listeners.DeploymentEventWrap, lister podLister) resourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.ReplicaSet, &v1beta1.ReplicaSet{}, eventsC, lister)
}

func newDaemonSetWatchLister(client rest.Interface, eventsC chan<- *listeners.DeploymentEventWrap, lister podLister) resourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.DaemonSet, &v1beta1.DaemonSet{}, eventsC, lister)
}

func newReplicationControllerWatchLister(client rest.Interface, eventsC chan<- *listeners.DeploymentEventWrap, lister podLister) resourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.ReplicationController, &v1.ReplicationController{}, eventsC, lister)
}

func newDeploymentWatcher(client rest.Interface, eventsC chan<- *listeners.DeploymentEventWrap, lister podLister) resourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.Deployment, &v1beta1.Deployment{}, eventsC, lister)
}

func newStatefulSetWatchLister(client rest.Interface, eventsC chan<- *listeners.DeploymentEventWrap, lister podLister) resourceWatchLister {
	return newReflectionWatcherFromClient(client, kubernetes.StatefulSet, &appsv1beta1.StatefulSet{}, eventsC, lister)
}
