package plan

import (
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/sensor/upgrader/common"
	appsV1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	deploymentGVK = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	daemonSetGVK  = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
)

func mergeResourceList(into *v1.ResourceList, from v1.ResourceList) {
	if *into == nil {
		*into = from.DeepCopy()
		return
	}
	*into = into.DeepCopy()
	for k, v := range from {
		(*into)[k] = v.DeepCopy()
	}
}

func applyOldResourcesConfig(newSpec, oldSpec *v1.PodSpec) {
	containerResourceReqs := make(map[string]*v1.ResourceRequirements)

	for i, ctr := range oldSpec.Containers {
		containerResourceReqs[ctr.Name] = &oldSpec.Containers[i].Resources
	}

	for i, ctr := range newSpec.Containers {
		oldReqs := containerResourceReqs[ctr.Name]
		if oldReqs == nil {
			continue
		}
		mergeResourceList(&newSpec.Containers[i].Resources.Requests, oldReqs.Requests)
		mergeResourceList(&newSpec.Containers[i].Resources.Limits, oldReqs.Limits)
	}
}

func getPodSpec(scheme *runtime.Scheme, obj k8sutil.Object) (k8sutil.Object, *v1.PodSpec, error) {
	var newObj k8sutil.Object
	switch obj.GetObjectKind().GroupVersionKind() {
	case deploymentGVK:
		if _, ok := obj.(*appsV1.Deployment); !ok {
			newObj = &appsV1.Deployment{}
		}
	case daemonSetGVK:
		if _, ok := obj.(*appsV1.DaemonSet); !ok {
			newObj = &appsV1.DaemonSet{}
		}
	default:
		return nil, nil, errors.Errorf("workload object of type %T with GVK %v is not recognized", obj, obj.GetObjectKind().GroupVersionKind())
	}

	if newObj != nil {
		if err := scheme.Convert(obj, newObj, nil); err != nil {
			return nil, nil, errors.Wrapf(err, "failed to convert workload object of type %T with GVK %v to strongly typed", obj, obj.GetObjectKind().GroupVersionKind())
		}
		obj = newObj
	}

	switch o := obj.(type) {
	case *appsV1.Deployment:
		return o, &o.Spec.Template.Spec, nil
	case *appsV1.DaemonSet:
		return o, &o.Spec.Template.Spec, nil
	default:
		return nil, nil, errors.Errorf("workload object of type %T with GVK %v is not recognized", obj, obj.GetObjectKind().GroupVersionKind())
	}
}

func applyPreservedProperties(scheme *runtime.Scheme, newObj, oldObj k8sutil.Object) (k8sutil.Object, error) {
	if oldObj.GetAnnotations()[common.PreserveResourcesAnnotationKey] != "true" {
		return newObj, nil
	}

	newAnns := newObj.GetAnnotations()
	if newAnns == nil {
		newAnns = make(map[string]string)
	}
	newAnns[common.PreserveResourcesAnnotationKey] = "true"
	newObj.SetAnnotations(newAnns)

	_, oldPodSpec, err := getPodSpec(scheme, oldObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract pod spec from old object")
	}
	newObjWithPodSpec, newPodSpec, err := getPodSpec(scheme, newObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract pod spec from new object")
	}

	applyOldResourcesConfig(newPodSpec, oldPodSpec)

	return newObjWithPodSpec, nil
}
