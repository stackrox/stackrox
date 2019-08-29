package upgradectx

import (
	"reflect"

	"github.com/stackrox/rox/sensor/upgrader/k8sobjects"
	"k8s.io/apimachinery/pkg/runtime"
)

func unpackListReflect(listObj runtime.Object) ([]k8sobjects.Object, bool) {
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
	result := make([]k8sobjects.Object, 0, l)
	for i := 0; i < l; i++ {
		item := itemsVal.Index(i).Addr()
		obj, _ := item.Interface().(k8sobjects.Object)
		if obj == nil {
			return nil, false
		}
		result = append(result, obj)
	}
	return result, true
}
