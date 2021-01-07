package plan

import (
	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// convert converts objects, adequately transferring type metadata.
func convert(scheme *runtime.Scheme, oldObj k8sutil.Object, newObj k8sutil.Object) error {
	if err := scheme.Convert(oldObj, newObj, nil); err != nil {
		return err
	}
	if newObj.GetObjectKind().GroupVersionKind() == (schema.GroupVersionKind{}) {
		newObj.GetObjectKind().SetGroupVersionKind(oldObj.GetObjectKind().GroupVersionKind())
	}
	return nil
}
