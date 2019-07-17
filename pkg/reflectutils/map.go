package reflectutils

import "reflect"

// ToTypedMap converts the given generic map (from interface{} to interface{}) to a typed map with the given key and
// value types.
func ToTypedMap(genericMap map[interface{}]interface{}, keyTy reflect.Type, valueTy reflect.Type) interface{} {
	mapTy := reflect.MapOf(keyTy, valueTy)
	resultMap := reflect.MakeMapWithSize(mapTy, len(genericMap))
	for k, v := range genericMap {
		resultMap.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	}
	return resultMap.Interface()
}
