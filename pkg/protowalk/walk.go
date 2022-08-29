package protowalk

import (
	"reflect"
)

func WalkProto(ty reflect.Type, fieldPath FieldPath, callback func(FieldPath) bool) {
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
			WalkProto(wrapperTy, fieldPath, callback)
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
		WalkProto(nextPath.Field().ElemType(), nextPath, callback)
	}
}
