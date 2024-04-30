package protoreflect

import (
	"reflect"

	"google.golang.org/protobuf/proto"
)

func IsProtoMessage(msgType reflect.Type) bool {
	_, ok := reflect.New(msgType).Interface().(proto.Message)

	return ok
}

func IsInternalGeneratorField(structField reflect.StructField) bool {
	return structField.Tag.Get("protobuf") == "" && structField.Tag.Get("protobuf_oneof") == ""
}
