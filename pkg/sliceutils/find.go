package sliceutils

import (
	"fmt"
	"reflect"
)

// Find returns the index of elem in slice, or -1 if slice does not contain elem.
func Find(slice interface{}, elem interface{}) int {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice {
		panic(fmt.Errorf("value is not of slice kind but %v", val.Kind()))
	}
	l := val.Len()
	for i := 0; i < l; i++ {
		if val.Index(i).Interface() == elem {
			return i
		}
	}
	return -1
}

// FindMatching returns the first index of slice where the passed predicate -- which must be a
// func(elemType) bool OR a func(*elemType) bool -- returns true, or -1 if it doesn't return true for any element.
// Example usage:
//
//	FindMatching([]string{"a", "b", "cd"}, func(s string) bool {
//	  return len(s) > 1
//	})
//
// will return 2.
// Note that the predicate could also be a func(s *string) bool if you want to avoid copying.
// This function will automatically pass pointers to each slice element if you pass such a predicate.
// It uses reflect, and will be slow.
// It panics at runtime if the arguments are of invalid types. There is no compile-time
// safety of any kind.
// Use ONLY in program initialization blocks, and in tests.
func FindMatching(slice interface{}, predicate interface{}) int {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() != reflect.Slice {
		panic(fmt.Errorf("FindMatching: value is not of slice kind but %v", sliceVal.Kind()))
	}

	predVal := reflect.ValueOf(predicate)
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
	for i := 0; i < l; i++ {
		valToUse := sliceVal.Index(i)
		if funcNeedsPtr {
			valToUse = valToUse.Addr()
		}
		out := predVal.Call([]reflect.Value{valToUse})
		if out[0].Bool() {
			return i
		}
	}
	return -1
}
