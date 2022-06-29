package sliceutils

import (
	"fmt"
	"reflect"
)

// Map maps the elements of slice, using the given mapFunc, which MUST be a
// func(elemType) returnElemType OR a func(*elemType) returnElemType.
// It returns a []returnElemType.
// Example usage:
// Map([]string{"a", "b", "cd"}, func(s string) int {
//   return len(s)
// })
// will return []int{1, 1, 2}.
// Note that the predicate could also be a func(s *string) int if you want to avoid copying.
// This function will automatically pass pointers to each slice element if you pass such a function.
// It uses reflect, and will be slow.
// It panics at runtime if the arguments are of invalid types. There is no compile-time
// safety of any kind.
// Use ONLY in program initialization blocks, and in tests.
func Map(slice, mapFunc interface{}) interface{} {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() != reflect.Slice {
		panic(fmt.Errorf("FindMatching: value is not of slice kind but %v", sliceVal.Kind()))
	}

	predVal := reflect.ValueOf(mapFunc)
	if predVal.Kind() != reflect.Func {
		panic(fmt.Errorf("FindMatching: value is not of predicate kind but %v", predVal.Kind()))
	}

	predType := predVal.Type()
	if predType.NumIn() != 1 {
		panic(fmt.Errorf("FindMatching: expected func to be unary but it was %d-ary", predType.NumIn()))
	}
	if predType.NumOut() != 1 {
		panic(fmt.Errorf("FindMatching: expected func to have exactly one return value, but it had %d", predType.NumOut()))
	}

	funcNeedsPtr := reflect.PtrTo(sliceVal.Type().Elem()).AssignableTo(predType.In(0))

	l := sliceVal.Len()
	outSlice := reflect.MakeSlice(reflect.SliceOf(predType.Out(0)), l, l)
	for i := 0; i < l; i++ {
		valToUse := sliceVal.Index(i)
		if funcNeedsPtr {
			valToUse = valToUse.Addr()
		}
		outSlice.Index(i).Set(predVal.Call([]reflect.Value{valToUse})[0])
	}
	return outSlice.Interface()
}

// MapsIntersect returns true there is at least one key-value pair that is present in both maps
// If both, or either maps are empty, it returns false
// TODO : Implement this so that it can take map[interface{}]interface{}
func MapsIntersect(m1 map[string]string, m2 map[string]string) bool {
	for k, v := range m1 {
		if val, exists := m2[k]; exists {
			if reflect.DeepEqual(v, val) {
				return true
			}
		}
	}
	return false
}
