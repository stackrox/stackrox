package listener

import (
	"reflect"
	"strings"

	pkgV1 "bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/pkg/api/v1"
	appsV1Beta1 "k8s.io/client-go/pkg/apis/apps/v1beta1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
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

	metaC chan resourceWrap
}

func newReflectionWatcherFromClient(client rest.Interface, resourceType string, objectType runtime.Object, metaC chan resourceWrap) *reflectionWatchLister {
	return newReflectionWatcher(newWatchLister(client), resourceType, objectType, metaC)
}

func newReflectionWatcher(watchLister watchLister, resourceType string, objectType runtime.Object, metaC chan resourceWrap) *reflectionWatchLister {
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
		metaC:          metaC,
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
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	meta, ok := objValue.FieldByIndex(wl.metaFieldIndex).Interface().(metav1.ObjectMeta)
	if !ok {
		logger.Errorf("obj %+v does not have an ObjectMeta field of the correct type", obj)
		return
	}

	// Ignore resources that are owned by another resource.
	if len(meta.OwnerReferences) > 0 {
		return
	}

	images := wl.getImages(objValue)
	replicas := wl.getReplicas(objValue)

	wl.metaC <- resourceWrap{
		ObjectMeta:   meta,
		resourceType: wl.resourceType(),
		replicas:     replicas,
		images:       images,
		action:       action,
	}
}

func (wl *reflectionWatchLister) getImages(objValue reflect.Value) (images []string) {
	spec := objValue.FieldByName("Spec")
	if reflect.DeepEqual(spec, reflect.Value{}) {
		logger.Errorf("Obj %+v does not have a Spec field", objValue)
		return
	}

	template, ok := spec.FieldByName("Template").Interface().(v1.PodTemplateSpec)
	if !ok {
		logger.Errorf("Spec obj %+v does not have a Template field", spec)
		return
	}

	for _, c := range template.Spec.Containers {
		images = append(images, c.Image)
	}

	return
}

func (wl *reflectionWatchLister) getReplicas(objValue reflect.Value) int {
	spec := objValue.FieldByName("Spec")
	if reflect.DeepEqual(spec, reflect.Value{}) {
		logger.Errorf("Obj %+v does not have a Spec field", objValue)
		return 0
	}

	replicaField := spec.FieldByName("Replicas")
	if reflect.DeepEqual(replicaField, reflect.Value{}) {
		return 0
	}

	replicasPointer, ok := replicaField.Interface().(*int32)
	if ok && replicasPointer != nil {
		return int(*replicasPointer)
	}

	replicas, ok := replicaField.Interface().(int32)
	if ok {
		return int(replicas)
	}

	return 0
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

func newReplicaSetWatchLister(client rest.Interface, metaC chan resourceWrap) resourceWatchLister {
	return newReflectionWatcherFromClient(client, replicaSet, &v1beta1.ReplicaSet{}, metaC)
}

func newDaemonSetWatchLister(client rest.Interface, metaC chan resourceWrap) resourceWatchLister {
	return newReflectionWatcherFromClient(client, daemonSet, &v1beta1.DaemonSet{}, metaC)
}

func newReplicationControllerWatchLister(client rest.Interface, metaC chan resourceWrap) resourceWatchLister {
	return newReflectionWatcherFromClient(client, replicationController, &v1.ReplicationController{}, metaC)
}

func newDeploymentWatcher(client rest.Interface, metaC chan resourceWrap) resourceWatchLister {
	return newReflectionWatcherFromClient(client, deployment, &v1beta1.Deployment{}, metaC)
}

func newStatefulSetWatchLister(client rest.Interface, metaC chan resourceWrap) resourceWatchLister {
	return newReflectionWatcherFromClient(client, statefulSet, &appsV1Beta1.StatefulSet{}, metaC)
}
