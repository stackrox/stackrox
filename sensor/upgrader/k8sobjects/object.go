package k8sobjects

import (
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Object is a combination of `runtime.Object` and `metav1.Object`.
type Object interface {
	runtime.Object
	metav1.Object
}

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
func RefOf(obj Object) ObjectRef {
	return ObjectRef{
		GVK:       obj.GetObjectKind().GroupVersionKind(),
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}
}

// BuildObjectMap takes a slice of Objects, and returns a map keyed by object reference.
func BuildObjectMap(objects []Object) map[ObjectRef]Object {
	result := make(map[ObjectRef]Object, len(objects))
	for _, obj := range objects {
		result[RefOf(obj)] = obj
	}
	return result
}
