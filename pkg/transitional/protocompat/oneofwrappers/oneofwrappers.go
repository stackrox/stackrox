package oneofwrappers

import (
	"reflect"

	"google.golang.org/protobuf/proto"
)

func OneofWrappers(m interface{}) []interface{} {
	if oneofWrappers, ok := m.(interface{ XXX_OneofWrappers() []interface{} }); ok {
		return oneofWrappers.XXX_OneofWrappers()
	}

	protoMsg := m.(proto.Message)
	t := reflect.TypeOf(m)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		panic("not a struct")
	}

	// Collect oneof fields on the Go level
	oneofFields := make(map[string]reflect.StructField)
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		oneofTag := f.Tag.Get("protobuf_oneof")
		if oneofTag == "" {
			continue
		}
		oneofFields[oneofTag] = f
	}

	if len(oneofFields) == 0 {
		return nil
	}

	msgDesc := protoMsg.ProtoReflect().Descriptor()
	msgCopy := protoMsg.ProtoReflect().New()
	msgCopyReflect := reflect.ValueOf(msgCopy.Interface()).Elem()

	oneofWrappers := make([]interface{}, 0, len(oneofFields))
	for i := 0; i < msgDesc.Fields().Len(); i++ {
		fieldDesc := msgDesc.Fields().Get(i)
		oneofDesc := fieldDesc.ContainingOneof()
		if oneofDesc == nil || oneofDesc.IsSynthetic() {
			continue // not a oneof field (or a proto3 optional field)
		}
		msgCopy.Set(fieldDesc, msgCopy.NewField(fieldDesc))
		fieldVal := msgCopyReflect.FieldByIndex(oneofFields[string(oneofDesc.Name())].Index)
		oneofWrappers = append(oneofWrappers, reflect.Zero(fieldVal.Elem().Type()).Interface())
	}

	return oneofWrappers
}
