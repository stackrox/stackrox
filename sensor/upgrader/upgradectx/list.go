package upgradectx

import (
	"reflect"

	"github.com/stackrox/stackrox/pkg/k8sutil"
	"k8s.io/apimachinery/pkg/runtime"
)

func unpackListReflect(listObj runtime.Object) ([]k8sutil.Object, bool) {
	listObjVal := reflect.ValueOf(listObj)
	if listObjVal.Kind() == reflect.Ptr {
		listObjVal = listObjVal.Elem()
	}
	if listObjVal.Kind() != reflect.Struct {
		return nil, false
	}
	itemsVal := listObjVal.FieldByName("Items")
	if itemsVal.Kind() != reflect.Slice {
		return nil, false
	}

	l := itemsVal.Len()
	result := make([]k8sutil.Object, 0, l)
	for i := 0; i < l; i++ {
		item := itemsVal.Index(i).Addr()
		obj, _ := item.Interface().(k8sutil.Object)
		if obj == nil {
			return nil, false
		}
		result = append(result, obj)
	}
	return result, true
}
