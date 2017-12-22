package listener

import (
	"reflect"
	"strings"

	pkgV1 "bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

const (
	deployment            = `Deployment`
	daemonSet             = `DaemonSet`
	replicationController = `ReplicationController`
	replicaSet            = `ReplicaSet`
	statefulSet           = `StatefulSet`
)

type resourceWatchLister interface {
	watch()
	stop()
	listObjects() []metav1.ObjectMeta
	resourceType() string
}

// A reflectionWatchLister extracts the ObjectMetadata using reflection.
type reflectionWatchLister struct {
	watchLister
	rt             string
	objectType     runtime.Object
	metaFieldIndex []int

	eventC chan<- *pkgV1.DeploymentEvent
}

func newReflectionWatcherFromClient(client rest.Interface, resourceType string, objectType runtime.Object, eventC chan<- *pkgV1.DeploymentEvent) *reflectionWatchLister {
	return newReflectionWatcher(newWatchLister(client), resourceType, objectType, eventC)
}

func newReflectionWatcher(watchLister watchLister, resourceType string, objectType runtime.Object, eventC chan<- *pkgV1.DeploymentEvent) *reflectionWatchLister {
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
		eventC:         eventC,
	}
}

func (wl *reflectionWatchLister) watch() {
	// We use the lowercase'd version of the resource type plus a plural "s" as the type of objects to watch.
	wl.watchLister.watch(strings.ToLower(wl.rt)+"s", wl.objectType, wl.resourceChanged)
}

func (wl *reflectionWatchLister) resourceType() string {
	return wl.rt
}

func (wl *reflectionWatchLister) stop() {
	wl.watchLister.stop()
}

func (wl *reflectionWatchLister) resourceChanged(obj interface{}, action pkgV1.ResourceAction) {
	if d := newDeploymentEventFromResource(obj, action, wl.metaFieldIndex, wl.resourceType()); d != nil {
		wl.eventC <- d
	}
}

func (wl *reflectionWatchLister) listObjects() (objects []metav1.ObjectMeta) {
	for _, obj := range wl.store.List() {
		objValue := reflect.Indirect(reflect.ValueOf(obj))
		meta, ok := objValue.FieldByIndex(wl.metaFieldIndex).Interface().(metav1.ObjectMeta)
		if !ok {
			logger.Errorf("obj %+v does not have an ObjectMeta field of the correct type", obj)
			continue
		}
		objects = append(objects, meta)
	}
	return
}

// Factory methods for the types of resources we support.

func newReplicaSetWatchLister(client rest.Interface, eventsC chan<- *pkgV1.DeploymentEvent) resourceWatchLister {
	return newReflectionWatcherFromClient(client, replicaSet, &v1beta1.ReplicaSet{}, eventsC)
}

func newDaemonSetWatchLister(client rest.Interface, eventsC chan<- *pkgV1.DeploymentEvent) resourceWatchLister {
	return newReflectionWatcherFromClient(client, daemonSet, &v1beta1.DaemonSet{}, eventsC)
}

func newReplicationControllerWatchLister(client rest.Interface, eventsC chan<- *pkgV1.DeploymentEvent) resourceWatchLister {
	return newReflectionWatcherFromClient(client, replicationController, &v1.ReplicationController{}, eventsC)
}

func newDeploymentWatcher(client rest.Interface, eventsC chan<- *pkgV1.DeploymentEvent) resourceWatchLister {
	return newReflectionWatcherFromClient(client, deployment, &v1beta1.Deployment{}, eventsC)
}

func newStatefulSetWatchLister(client rest.Interface, eventsC chan<- *pkgV1.DeploymentEvent) resourceWatchLister {
	return newReflectionWatcherFromClient(client, statefulSet, &appsv1beta1.StatefulSet{}, eventsC)
}
