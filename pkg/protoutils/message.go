package protoutils

import (
	"reflect"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// MessageType is a replacement for deprecated github.com/golang/protobuf/proto.MessageType
func MessageType(typ string) reflect.Type {
	mt, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typ))
	if err != nil {
		return nil
	}
	return reflect.TypeOf(mt.Zero().Interface())
}
