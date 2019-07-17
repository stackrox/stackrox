package reflectutils

import "reflect"

// ToTypedSlice converts the given generic (interface{}) slice to a slice with the given element types.
func ToTypedSlice(genericSlice []interface{}, elemTy reflect.Type) interface{} {
	sliceTy := reflect.SliceOf(elemTy)
	slice := reflect.MakeSlice(sliceTy, len(genericSlice), len(genericSlice))
	for i, elem := range genericSlice {
		slice.Index(i).Set(reflect.ValueOf(elem))
	}
	return slice.Interface()
}
