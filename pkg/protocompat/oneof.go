package protocompat

import (
	"reflect"

	"github.com/gogo/protobuf/proto"
)

func GetOneOfFieldTypes(msgType reflect.Type, fieldIndex int) []reflect.Type {
	oneOfFieldTypes := make([]reflect.Type, 0)

	structProps := proto.GetProperties(msgType)
	for _, oneOfField := range structProps.OneofTypes {
		if oneOfField.Field != fieldIndex {
			continue
		}

		if !oneOfField.Type.Implements(msgType.Field(fieldIndex).Type) {
			continue
		}

		oneOfFieldTypes = append(oneOfFieldTypes, oneOfField.Type)
	}

	return oneOfFieldTypes
}
