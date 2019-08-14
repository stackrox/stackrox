package sliceutils

import (
	"fmt"
	"reflect"
)

func reverseInPlace(sliceVal reflect.Value, l int) {
	for i := 0; i < l/2; i++ {
		val1, val2 := sliceVal.Index(i), sliceVal.Index(l-1-i)
		tmp := val1.Interface()
		val1.Set(val2)
		val2.Set(reflect.ValueOf(tmp))
	}
}

// ReverseInPlace reverses the elements of the given slice in-place.
func ReverseInPlace(slice interface{}) {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice {
		panic(fmt.Errorf("value is not of slice kind but %v", val.Kind()))
	}
	l := val.Len()
	reverseInPlace(val, l)
}

// Reversed returns a slice that contains the elements of the input slice in reverse order.
func Reversed(slice interface{}) interface{} {
	val := reflect.ValueOf(slice)
	if val.Kind() != reflect.Slice {
		panic(fmt.Errorf("value is not of slice kind but %v", val.Kind()))
	}

	l := val.Len()
	out := reflect.MakeSlice(val.Type(), l, l)
	reflect.Copy(out, val)
	reverseInPlace(out, l)
	return out.Interface()
}
