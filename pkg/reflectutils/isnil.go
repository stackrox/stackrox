package reflectutils

import (
	"reflect"
)

// IsNil uses reflection to reliably check if the provided argument is a Nil pointer.
func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Pointer, reflect.Map, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
