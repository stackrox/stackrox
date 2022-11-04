package logic

import "reflect"

// Truthy returns the truthiness value of an arbitary value. The nil interface and zero values are always falsy.
// Empty slices and maps are falsy as well, even if they are non-nil. All other values are truthy.
func Truthy(val interface{}) bool {
	if val == nil {
		return false
	}
	rval := reflect.ValueOf(val)
	if rval.IsZero() {
		return false
	}
	if rval.Kind() == reflect.Slice || rval.Kind() == reflect.Map {
		return rval.Len() > 0
	}
	return true
}
