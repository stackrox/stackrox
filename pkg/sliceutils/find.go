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
