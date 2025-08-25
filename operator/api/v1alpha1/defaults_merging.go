package v1alpha1

import (
	"reflect"
)

type nonDereferencingLeafTransformer struct{}

func (t nonDereferencingLeafTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	// Only handle pointer types
	if typ.Kind() == reflect.Ptr {
		elem := typ.Elem()

		// If pointer to struct: return no transformer function so that mergo continues traversing.
		// Retain mergo's merging behaviour for maps and slices.
		if elem.Kind() == reflect.Struct || elem.Kind() == reflect.Map || elem.Kind() == reflect.Slice {
			return nil
		}

		// Use custom transformer function for other (leaf) types.
		return func(dst, src reflect.Value) error {
			if dst.IsNil() && !src.IsNil() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}
