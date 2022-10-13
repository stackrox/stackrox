package plan

import (
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/k8sutil/k8sobjects"
	"github.com/stackrox/rox/sensor/upgrader/common"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	collectorName = "collector"
)

var (
	deploymentGVK = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	daemonSetGVK  = schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"}
)

func mergeResourceList(into *map[string]interface{}, from map[string]interface{}) {
	if *into == nil {
		*into = runtime.DeepCopyJSON(from)
		return
	}
	*into = runtime.DeepCopyJSON(*into)
	for k, v := range from {
		(*into)[k] = runtime.DeepCopyJSONValue(v)
	}
}

func applyOldResourcesConfig(newPodSpec, oldPodSpec map[string]interface{}) error {
	containerResourceReqs := make(map[string]map[string]interface{})

	oldContainers, err := nestedValueNoCopyOrError[[]interface{}](oldPodSpec, "containers")
	if err != nil {
		return errors.Wrap(err, "retrieving containers field from old pod spec")
	}
	for _, ctrRaw := range oldContainers {
		ctr, _ := ctrRaw.(map[string]interface{})
		if ctr == nil {
			return errors.New("non-map entry in old pod spec containers")
		}
		ctrName, err := nestedNonZeroValueNoCopyOrError[string](ctr, "name")
		if err != nil {
			return errors.Wrap(err, "getting container name in old pod spec")
		}
		resources := nestedValueNoCopyOrDefault[map[string]interface{}](ctr, nil, "resources")
		if resources != nil {
			containerResourceReqs[ctrName] = resources
		}
	}

	newContainers, err := nestedValueNoCopyOrError[[]interface{}](newPodSpec, "containers")
	if err != nil {
		return errors.Wrap(err, "retrieving containers field from new pod spec")
	}
	for _, ctrRaw := range newContainers {
		ctr, _ := ctrRaw.(map[string]interface{})
		if ctr == nil {
			return errors.New("non-map entry in new pod spec containers")
		}
		ctrName, err := nestedNonZeroValueNoCopyOrError[string](ctr, "name")
		if err != nil {
			return errors.Wrap(err, "getting container name in new pod spec")
		}
		oldResources := containerResourceReqs[ctrName]
		if oldResources == nil {
			continue
		}
		newResources := nestedValueNoCopyOrDefault[map[string]interface{}](ctr, nil, "resources")
		if newResources == nil {
			newResources = make(map[string]interface{})
		}
		oldRequests := nestedValueNoCopyOrDefault[map[string]interface{}](oldResources, nil, "requests")
		if oldRequests != nil {
			newRequests := nestedValueNoCopyOrDefault[map[string]interface{}](newResources, nil, "requests")
			mergeResourceList(&newRequests, oldRequests)
			if err := unstructured.SetNestedField(newResources, newRequests, "requests"); err != nil {
				return errors.Wrapf(err, "could not set new resource requests for container %s", ctrName)
			}
		}
		oldLimits := nestedValueNoCopyOrDefault[map[string]interface{}](oldResources, nil, "limits")
		if oldLimits != nil {
			newLimits := nestedValueNoCopyOrDefault[map[string]interface{}](newResources, nil, "limits")
			mergeResourceList(&newLimits, oldLimits)
			if err := unstructured.SetNestedField(newResources, newLimits, "limits"); err != nil {
				return errors.Wrapf(err, "could not set new resource limits for container %s", ctrName)
			}
		}
		if err := unstructured.SetNestedField(ctr, newResources, "resources"); err != nil {
			return errors.Wrapf(err, "could not set new resources for container %s", ctrName)
		}
	}
	return nil
}

func getPodSpec(obj *unstructured.Unstructured) (map[string]interface{}, error) {
	var podSpecPath []string
	switch obj.GetObjectKind().GroupVersionKind() {
	case deploymentGVK, daemonSetGVK:
		podSpecPath = []string{"spec", "template", "spec"}
	default:
		return nil, errors.Errorf("workload object of type %T with GVK %v is not recognized", obj, obj.GetObjectKind().GroupVersionKind())
	}

	podSpecRaw, _, err := unstructured.NestedFieldNoCopy(obj.Object, podSpecPath...)
	if err != nil {
		return nil, errors.Wrapf(err, "locating pod spec in object %s", k8sobjects.RefOf(obj))
	}
	podSpec, _ := podSpecRaw.(map[string]interface{})
	if podSpec == nil {
		return nil, errors.Errorf("did not find pod spec in object %s", k8sobjects.RefOf(obj))
	}
	return podSpec, nil
}

func applyPreservedResources(newObj, oldObj *unstructured.Unstructured) error {
	newAnns := newObj.GetAnnotations()
	if newAnns == nil {
		newAnns = make(map[string]string)
	}
	newAnns[common.PreserveResourcesAnnotationKey] = "true"
	newObj.SetAnnotations(newAnns)

	oldPodSpec, err := getPodSpec(oldObj)
	if err != nil {
		return errors.Wrap(err, "failed to extract pod spec from old object")
	}
	newPodSpec, err := getPodSpec(newObj)
	if err != nil {
		return errors.Wrap(err, "failed to extract pod spec from new object")
	}

	if err := applyOldResourcesConfig(newPodSpec, oldPodSpec); err != nil {
		return errors.Wrap(err, "failed to preserve resources")
	}

	return nil
}

func applyPreservedTolerations(newObj, oldObj *unstructured.Unstructured) error {
	oldPodSpec, err := getPodSpec(oldObj)
	if err != nil {
		return errors.Wrap(err, "failed to extract pod spec from old object")
	}
	newPodSpec, err := getPodSpec(newObj)
	if err != nil {
		return errors.Wrap(err, "failed to extract pod spec from new object")
	}

	if tolerations := nestedValueNoCopyOrDefault[[]interface{}](oldPodSpec, nil, "tolerations"); tolerations != nil {
		if err := unstructured.SetNestedField(newPodSpec, tolerations, "tolerations"); err != nil {
			return errors.Wrap(err, "failed to preserve tolerations from old pod spec")
		}
	}
	return nil
}

func applyServicePreservedProperties(newObj, oldObj *unstructured.Unstructured) error {
	clusterIP := nestedValueNoCopyOrDefault[string](oldObj.Object, "", "spec", "clusterIP")
	if clusterIP != "" {
		if err := unstructured.SetNestedField(newObj.Object, clusterIP, "spec", "clusterIP"); err != nil {
			return errors.Wrap(err, "setting cluster IP")
		}
	}
	return nil
}

func applyPreservedProperties(newObj, oldObj *unstructured.Unstructured) error {
	var overallErr *multierror.Error
	if newObj.GetObjectKind().GroupVersionKind() == serviceGVK {
		if err := applyServicePreservedProperties(newObj, oldObj); err != nil {
			overallErr = multierror.Append(overallErr, errors.Wrap(err, "failed to preserve service properties"))
		}
	}
	if oldObj.GetAnnotations()[common.PreserveResourcesAnnotationKey] == "true" {
		if err := applyPreservedResources(newObj, oldObj); err != nil {
			overallErr = multierror.Append(overallErr, errors.Wrap(err, "failed to preserve resources"))
		}
	}

	switch newObj.GetObjectKind().GroupVersionKind() {
	case deploymentGVK, daemonSetGVK:
	default:
		return overallErr.ErrorOrNil()
	}

	// Ignore collector because tolerations are explicitly set
	if newObj.GetObjectKind().GroupVersionKind() == daemonSetGVK && newObj.GetName() == collectorName {
		return overallErr.ErrorOrNil()
	}

	if err := applyPreservedTolerations(newObj, oldObj); err != nil {
		overallErr = multierror.Append(overallErr, errors.Wrap(err, "failed to preserve tolerations"))
	}
	return overallErr.ErrorOrNil()
}
