package protocompat

import (
	"reflect"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/runtime/protoimpl"
)

func oneOfFieldTypeCmp(a reflect.Type, b reflect.Type) int {
	if a.Kind() != reflect.Ptr || b.Kind() != reflect.Ptr {
		return 0
	}

	return strings.Compare(a.Elem().Name(), b.Elem().Name())
}

func GetOneOfTypesByFieldIndex(msgType reflect.Type, fieldIndex int) []reflect.Type {
	return GetOneOfTypesByInterface(msgType, msgType.Field(fieldIndex).Type)
}

func GetOneOfTypesByInterface(msgType reflect.Type, oneOfInterfaceType reflect.Type) []reflect.Type {
	oneOfFieldTypes := make([]reflect.Type, 0)

	// Get proto message from Go reflect type.
	msg, ok := reflect.New(msgType).Interface().(proto.Message)
	if !ok {
		return oneOfFieldTypes
	}

	// Get proto reflection information from proto message. We need to cast to
	// protoimpl.MessageInfo, because protoreflect.MessageType interface
	// does not provide way to get OneofWrappers.
	msgInfo, ok := msg.ProtoReflect().Type().(*protoimpl.MessageInfo)
	if !ok {
		return oneOfFieldTypes
	}

	for _, oneOfWrapper := range msgInfo.OneofWrappers {
		typ := reflect.TypeOf(oneOfWrapper)
		if typ.Implements(oneOfInterfaceType) {
			oneOfFieldTypes = append(oneOfFieldTypes, typ)
		}
	}

	slices.SortFunc(oneOfFieldTypes, oneOfFieldTypeCmp)

	return oneOfFieldTypes
}
