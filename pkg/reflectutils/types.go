package reflectutils

import (
	"reflect"
)

// Type returns the type of the interface in string format
func Type(i any) string {
	return reflect.TypeOf(i).String()
}
