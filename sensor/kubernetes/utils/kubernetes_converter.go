package utils

import (
	"reflect"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// FromUnstructuredToSpecificTypePointer converts the unstructured object 'from' into the specific type 'to'
func FromUnstructuredToSpecificTypePointer(from any, to any) error {
	if reflect.ValueOf(to).Kind() != reflect.Ptr {
		return errors.Errorf("expected 'to' to be a pointer, but got: '%T'", to)
	}
	unstructuredObj, ok := from.(*unstructured.Unstructured)
	if !ok {
		return errors.Errorf("expected 'from' to be of type 'Unstructured' but got: %T", from)
	}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, to); err != nil {
		return errors.Wrapf(err, "unable to convert 'Unstructured' to '%T'", to)
	}
	return nil
}
