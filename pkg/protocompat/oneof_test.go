package protocompat

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
)

func testHelperGetFieldProtoName(field reflect.StructField) string {
	tagVal := field.Tag.Get("protobuf")
	if tagVal != "" {
		tagParts := strings.Split(tagVal, ",")
		for _, tagPart := range tagParts {
			if strings.HasPrefix(tagPart, "name=") {
				return strings.TrimLeft(tagPart, "name=")
			}
		}
	}

	return ""
}

func TestGetOneOfFieldTypes(t *testing.T) {
	msg := storage.TestSingleUUIDKeyStruct{}

	msgType := reflect.TypeOf(msg)
	assert.NotNil(t, msgType)

	fieldsToTest := map[string]set.StringSet{
		"Oneof":    set.NewStringSet("oneofstring", "oneofnested"),
		"OneofTwo": set.NewStringSet("oneof_two_string", "oneof_two_int"),
	}

	for oneOfFieldName, oneOfFieldNamesSet := range fieldsToTest {
		fieldType, err := msgType.FieldByName(oneOfFieldName)
		assert.NotNil(t, err)

		oneOfFieldSubTypes := GetOneOfFieldTypes(msgType, fieldType.Index[0])
		for _, subType := range oneOfFieldSubTypes {
			subTypeElem := subType.Elem()
			for subTypeFieldIndex := 0; subTypeFieldIndex < subTypeElem.NumField(); subTypeFieldIndex++ {
				protoFieldName := testHelperGetFieldProtoName(subTypeElem.Field(subTypeFieldIndex))

				assert.True(t, oneOfFieldNamesSet.Contains(protoFieldName), fmt.Sprintf("Field %q is not expected for %q type", protoFieldName, fieldType.Name))
				oneOfFieldNamesSet.Remove(protoFieldName)
			}
		}

		assert.Equal(t, 0, oneOfFieldNamesSet.Cardinality(), fmt.Sprintf("Not all oneof fields are found for %q field", fieldType.Name))
	}
}
