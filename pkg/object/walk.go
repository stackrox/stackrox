package object

import (
	"reflect"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/protoreflect"
)

// WalkObject generates a generic struct object that then can be walked by other code
func WalkObject(obj interface{}) Struct {
	return walk(Struct{}, reflect.StructField{}, reflect.TypeOf(obj)).(Struct)
}

func walk(s Struct, field reflect.StructField, typ reflect.Type) Field {
	kind := typ.Kind()
	switch kind {
	case reflect.Ptr:
		return walk(s, field, typ.Elem())
	case reflect.Struct:
		if typ == reflect.TypeOf(types.Timestamp{}) {
			return Struct{
				Field:      newField(field, kind),
				StructType: TIME,
			}
		}

		childStruct := Struct{
			Field:      newField(field, kind),
			StructType: MESSAGE,
		}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)

			if field.Tag.Get("protobuf_oneof") != "" {
				oneofStruct := Struct{
					Field:      newField(field, kind),
					StructType: ONEOF,
				}

				actualOneOfFields := reflect.New(typ).Interface().(interface{ XXX_OneofWrappers() []interface{} }).XXX_OneofWrappers()
				for _, f := range actualOneOfFields {
					typ := reflect.TypeOf(f)
					if typ.Implements(field.Type) {
						// Ignore the wrapper struct
						field := typ.Elem().Field(0)
						oneofStruct.Fields = append(oneofStruct.Fields, walk(oneofStruct, field, field.Type))
					}
				}
				childStruct.Fields = append(childStruct.Fields, oneofStruct)
				continue
			}
			childStruct.Fields = append(childStruct.Fields, walk(s, field, field.Type))
		}
		return childStruct
	case reflect.Slice:
		return Slice{
			Field: newField(field, kind),
			Value: walk(s, reflect.StructField{}, typ.Elem()),
		}
	case reflect.Map:
		return Map{
			Field: newField(field, kind),
			Value: walk(s, reflect.StructField{}, typ.Elem()),
		}
	case reflect.Int32:
		enum, ok := reflect.Zero(typ).Interface().(protoreflect.ProtoEnum)
		if !ok {
			return newField(field, kind)
		}
		desc, err := protoreflect.GetEnumDescriptor(enum)
		if err != nil {
			panic(err)
		}
		return Enum{
			Field:      newField(field, kind),
			Descriptor: desc,
		}
	case reflect.Interface:
		panic("shouldn't ever get to this")
	default:
		return newField(field, kind)
	}
}
