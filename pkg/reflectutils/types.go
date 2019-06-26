package reflectutils

import (
	"reflect"
)

// Type returns the type of the interface in string format
func Type(i interface{}) string {
	return reflect.TypeOf(i).String()
}
