package protoreflect

import "reflect"

func IsInternalGeneratorField(structField reflect.StructField) bool {
	return structField.Tag.Get("protobuf") == "" && structField.Tag.Get("protobuf_oneof") == ""
}
