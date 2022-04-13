package object

import (
	"reflect"
	"testing"

	"github.com/stackrox/stackrox/generated/test"
	"github.com/stretchr/testify/assert"
)

func checkCloneSubMessage(t *testing.T, field Field) {
	assert.Equal(t, field.Type(), reflect.Struct)

	str, ok := field.(Struct)
	assert.True(t, ok)
	assert.Equal(t, str.StructType, MESSAGE)

	checkBaseType(t, str.Fields[0], "Int32", reflect.Int32, "int32,omitempty")
	checkBaseType(t, str.Fields[1], "String_", reflect.String, "string,omitempty")
}

func checkBaseType(t *testing.T, field Field, name string, kind reflect.Kind, jsonTags string) {
	assert.Equal(t, field.Name(), name)
	assert.Equal(t, field.Type(), kind)
	assert.Equal(t, field.Tags().Get("json"), jsonTags)
}

func TestWalk(t *testing.T) {
	object := WalkObject((*test.TestClone)(nil))

	assert.Equal(t, object.StructType, MESSAGE)
	assert.Equal(t, object.Name(), "")
	assert.Equal(t, object.Type(), reflect.Struct)

	checkBaseType(t, object.Fields[0], "IntSlice", reflect.Slice, "int_slice,omitempty")
	checkBaseType(t, object.Fields[1], "StringSlice", reflect.Slice, "string_slice,omitempty")
	checkBaseType(t, object.Fields[2], "SubMessages", reflect.Slice, "sub_messages,omitempty")
	checkCloneSubMessage(t, object.Fields[2].(Slice).Value)

	checkBaseType(t, object.Fields[3], "MessageMap", reflect.Map, "message_map,omitempty")
	checkCloneSubMessage(t, object.Fields[3].(Map).Value)

	checkBaseType(t, object.Fields[4], "StringMap", reflect.Map, "string_map,omitempty")
	checkBaseType(t, object.Fields[4].(Map).Value, "", reflect.String, "")

	checkBaseType(t, object.Fields[5], "EnumSlice", reflect.Slice, "enum_slice,omitempty")
	checkBaseType(t, object.Fields[5].(Slice).Value.(Enum), "", reflect.Int32, "")

	checkBaseType(t, object.Fields[6], "Ts", reflect.Struct, "ts,omitempty")
	assert.Equal(t, object.Fields[6].(Struct).StructType, TIME)

	checkBaseType(t, object.Fields[7], "Primitive", reflect.Struct, "")
	assert.Equal(t, object.Fields[7].(Struct).StructType, ONEOF)
	oneof := object.Fields[7].(Struct)
	checkBaseType(t, oneof.Fields[0], "Int32", reflect.Int32, "int32,omitempty")
	checkBaseType(t, oneof.Fields[1], "String_", reflect.String, "string,omitempty")
	checkCloneSubMessage(t, oneof.Fields[2])
}
