package sliceutils

import (
	"fmt"
	"reflect"
)

// Unique returns a new slice that contains only the first occurrence of each element in slice.
// Callers should be able to cast the returned value to the same slice type that they passed in.
// Example: Unique([]string{"a", "a", b"}).([]string) will work.
// Use only in tests or initialization code, where the performance penalty
// and lack of compile-time safety are worth it.
// For all other cases, use code generation (see generic.go).
func Unique(slice interface{}) interface{} {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice {
		panic(fmt.Errorf("value is not of slice kind but %v", val.Kind()))
	}

	l := val.Len()
	out := reflect.MakeSlice(val.Type(), 0, l)

	seenElems := make(map[interface{}]struct{})
	for i := 0; i < l; i++ {
		elem := val.Index(i).Interface()
		if _, ok := seenElems[elem]; !ok {
			out = reflect.Append(out, reflect.ValueOf(elem))
			seenElems[elem] = struct{}{}
		}
	}

	return out.Interface()
}
