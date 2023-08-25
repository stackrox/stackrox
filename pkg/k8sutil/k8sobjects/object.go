package k8sobjects

import (
	"strings"

	"github.com/stackrox/rox/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ObjectRef references a Kubernetes object.
type ObjectRef struct {
	GVK       schema.GroupVersionKind
	Name      string
	Namespace string
}

// String returns a string representation of this object reference.
func (r ObjectRef) String() string {
	var b strings.Builder
	if r.Namespace != "" {
		b.WriteString(r.Namespace)
		b.WriteRune('/')
	}
	b.WriteString(r.Name)
	b.WriteRune('[')
	b.WriteString(r.GVK.String())
	b.WriteRune(']')
	return b.String()
}

// RefOf returns an ObjectRef for the given object.
func RefOf(obj k8sutil.Object) ObjectRef {
	return ObjectRef{
		GVK:       obj.GetObjectKind().GroupVersionKind(),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// BuildObjectMap takes a slice of Objects, and returns a map keyed by object reference.
func BuildObjectMap[T k8sutil.Object](objects []T) map[ObjectRef]T {
	result := make(map[ObjectRef]T, len(objects))
	for _, obj := range objects {
		result[RefOf(obj)] = obj
	}
	return result
}
