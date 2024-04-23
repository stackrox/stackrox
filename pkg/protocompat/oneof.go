package protocompat

import (
	"reflect"
	"slices"
	"strings"

	"github.com/gogo/protobuf/proto"
)

func oneOfFieldTypeCmp(a reflect.Type, b reflect.Type) int {
	if a.Kind() != reflect.Ptr || b.Kind() != reflect.Ptr {
		return 0
	}

	return strings.Compare(a.Elem().Name(), b.Elem().Name())
}

func GetOneOfTypesByFieldIndex(msgType reflect.Type, fieldIndex int) []reflect.Type {
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

	slices.SortFunc(oneOfFieldTypes, oneOfFieldTypeCmp)

	return oneOfFieldTypes
}

func GetOneOfTypesByInterface(msgType reflect.Type, oneOfInterfaceType reflect.Type) []reflect.Type {
	oneOfFieldTypes := make([]reflect.Type, 0)

	structProps := proto.GetProperties(msgType)
	for _, oneOfField := range structProps.OneofTypes {
		if !oneOfField.Type.Implements(oneOfInterfaceType) {
			continue
		}

		oneOfFieldTypes = append(oneOfFieldTypes, oneOfField.Type)
	}

	slices.SortFunc(oneOfFieldTypes, oneOfFieldTypeCmp)

	return oneOfFieldTypes
}
