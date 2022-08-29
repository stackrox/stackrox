package protowalk

import (
	"reflect"
)

// WalkProto traverses a protobuf message, calling the given callback for each traversed field.
// ty should be a pointer to a struct that represents a protobuf message. Fields are then traversed in depth-first
// order, with callback being invoked with the complete field path (from the root) for each field encountered during
// traversal.
// If callback returns false and the current field is not a leaf field, the descendant fields of that fields will not
// be visited. Note that in contrast to the hierarchy at the protobuf level, each alternative of a oneof is regarded
// as a descendant of the oneof "field".
func WalkProto(ty reflect.Type, callback func(FieldPath) bool) {
	walkProto(ty, nil, callback)
}

func walkProto(ty reflect.Type, fieldPath FieldPath, callback func(FieldPath) bool) {
	if len(fieldPath) > 0 {
		if !callback(fieldPath) {
			return
		}
	}

	if ty.Kind() == reflect.Interface {
		oneofWrappers := reflect.Zero(fieldPath.Field().ContainingType).Interface().(interface{ XXX_OneofWrappers() []interface{} }).XXX_OneofWrappers()
		for _, w := range oneofWrappers {
			wrapperTy := reflect.TypeOf(w)
			if !wrapperTy.Implements(ty) {
				continue
			}
			walkProto(wrapperTy, fieldPath, callback)
		}
		return
	}

	if ty.Kind() != reflect.Ptr || ty.Elem().Kind() != reflect.Struct {
		return
	}

	elemTy := ty.Elem()
	for i := 0; i < elemTy.NumField(); i++ {
		f := elemTy.Field(i)

		if f.Tag.Get("protobuf") == "" && f.Tag.Get("protobuf_oneof") == "" {
			continue // not a proto or oneof wrapper field
		}

		nextPath := append(fieldPath, Field{ContainingType: ty, StructField: f})
		walkProto(nextPath.Field().ElemType(), nextPath, callback)
	}
}
