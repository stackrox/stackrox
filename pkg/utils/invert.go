package utils

import (
	"fmt"
	"reflect"
)

var (
	intTy = reflect.TypeOf(0)
)

// Invert inverts an object, which must be a map, array, or slice, and returns a map mapping elements to keys (or
// indices). If multiple keys/indices map to the same element, the resulting mapping is non-deterministic.
// An error is returned if the given object is not of kind map, array, or slice.
// Note: since this function is based on reflection, it should not be used in performance-critical code. Its intended
// use is in global variable definitions and `init()` blocks. For this reason, there is no explicit error handling - all
// errors are reported as panics.
func Invert(obj interface{}) interface{} {
	objVal := reflect.ValueOf(obj)
	switch objVal.Kind() {
	case reflect.Slice, reflect.Array:
		return invertSliceOrArray(objVal)
	case reflect.Map:
		return invertMap(objVal)
	default:
		panic(fmt.Errorf("object is neither of kind slice, array, or map, but %v", objVal.Kind()))
	}
}

func invertSliceOrArray(sliceVal reflect.Value) interface{} {
	elemTy := sliceVal.Type().Elem()
	length := sliceVal.Len()
	mapTy := reflect.MapOf(elemTy, intTy)
	result := reflect.MakeMapWithSize(mapTy, length)
	for i := 0; i < length; i++ {
		elem := sliceVal.Index(i)
		result.SetMapIndex(elem, reflect.ValueOf(i))
	}
	return result.Interface()
}

func invertMap(mapVal reflect.Value) interface{} {
	length := mapVal.Len()
	keyTy, elemTy := mapVal.Type().Key(), mapVal.Type().Elem()
	mapTy := reflect.MapOf(elemTy, keyTy)
	result := reflect.MakeMapWithSize(mapTy, length)
	for _, keyVal := range mapVal.MapKeys() {
		elemVal := mapVal.MapIndex(keyVal)
		result.SetMapIndex(elemVal, keyVal)
	}
	return result.Interface()
}

// InvertList inverts the given variadic argument list, returning a map mapping elements to indices.
func InvertList(objs ...interface{}) interface{} {
	return Invert(objs)
}
