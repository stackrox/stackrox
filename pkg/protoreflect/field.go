package protoreflect

import (
	"reflect"

	"github.com/stackrox/rox/pkg/protocompat"
)

func IsProtoMessage(msgType reflect.Type) bool {
	_, ok := reflect.New(msgType).Interface().(protocompat.Message)

	return ok
}

func IsInternalGeneratorField(structField reflect.StructField) bool {
	return structField.Tag.Get("protobuf") == "" && structField.Tag.Get("protobuf_oneof") == ""
}
