package v1alpha1

import (
	"reflect"
)

type nonDereferencingLeafTransformer struct{}

func (t nonDereferencingLeafTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	// Only handle pointer types here.
	if typ.Kind() != reflect.Ptr {
		return nil
	}

	elemKind := typ.Elem().Kind()

	// If pointer to struct, map or slice: return no transformer function so that mergo continues traversing.
	if elemKind == reflect.Struct || elemKind == reflect.Map || elemKind == reflect.Slice {
		return nil
	}

	// Use custom transformer function for other (leaf) types: override nil fields with non-nil defaults.
	return func(dst, src reflect.Value) error {
		if dst.IsNil() && !src.IsNil() {
			dst.Set(src)
		}
		return nil
	}
}
