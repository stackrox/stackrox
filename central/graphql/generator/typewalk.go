package generator

import (
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoreflect"
)

type fieldData struct {
	Name string
	Type reflect.Type
}

type unionData struct {
	Name    string
	Entries []fieldData
}

type typeData struct {
	Name      string
	Package   string
	Type      reflect.Type
	FieldData []fieldData
	UnionData []unionData
}

type walkState struct {
	typeData  map[reflect.Type]typeData
	typeQueue []reflect.Type
}

func (ctx *walkState) walkUnions(p reflect.Type) (output []unionData) {
	msg := reflect.Zero(p).Interface().(protoreflect.ProtoMessage)
	des, err := protoreflect.GetMessageDescriptor(msg)
	if err != nil {
		panic(errors.Wrapf(err, "err on type %s", p.Name()))
	}
	for i, oneOf := range des.GetOneofDecl() {
		union := unionData{
			Name: generator.CamelCase(oneOf.GetName()),
		}
		for _, field := range des.GetField() {
			if field.OneofIndex == nil || *field.OneofIndex != int32(i) {
				continue
			}
			if field.GetType() != descriptor.FieldDescriptorProto_TYPE_MESSAGE {
				continue
			}
			// field.GetTypeName() has a leading . that nothing else expects
			msgTypeName := field.GetTypeName()
			if len(msgTypeName) > 1 && msgTypeName[0] == '.' {
				msgTypeName = msgTypeName[1:]
			}
			msgType := proto.MessageType(msgTypeName)
			if msgType == nil {
				continue
			}
			ctx.typeQueue = append(ctx.typeQueue, msgType)
			union.Entries = append(union.Entries, fieldData{
				Name: generator.CamelCase(field.GetName()),
				Type: msgType,
			})
		}
		if len(union.Entries) > 0 {
			output = append(output, union)
		}
	}
	sort.Slice(output, func(i, j int) bool {
		return output[i].Name < output[j].Name
	})
	return
}

func (ctx *walkState) walkType(p reflect.Type) {
	var unions []unionData
	if p.Kind() == reflect.Slice {
		p = p.Elem()
	}
	if p != nil && p.Implements(messageType) {
		unions = ctx.walkUnions(p)
	}
	for p.Kind() == reflect.Ptr {
		p = p.Elem()
	}
	if _, ok := ctx.typeData[p]; ok {
		return
	}
	td := typeData{Name: p.Name(), Package: p.PkgPath(), Type: p, UnionData: unions}
	if p.Kind() == reflect.Struct {
		for i := 0; i < p.NumField(); i++ {
			ctx.walkField(&td, p, p.Field(i))
		}
		sort.Slice(td.FieldData, func(i, j int) bool {
			return td.FieldData[i].Name < td.FieldData[j].Name
		})
	}
	ctx.typeData[p] = td
}

func (ctx *walkState) walkField(td *typeData, p reflect.Type, sf reflect.StructField) {
	if len(sf.Name) > 4 && sf.Name[:4] == "XXX_" {
		return
	}
	if strings.HasPrefix(sf.Name, "DEPRECATED") {
		return
	}
	ctx.typeQueue = append(ctx.typeQueue, sf.Type)
	td.FieldData = append(td.FieldData, fieldData{
		Name: sf.Name,
		Type: sf.Type,
	})
}

func rejected(p reflect.Type, blacklist []reflect.Type) bool {
	for _, t := range blacklist {
		if t == p {
			return true
		}
	}
	return false
}

func typeWalk(initial []reflect.Type, blacklist []reflect.Type) []typeData {
	ctx := walkState{
		typeData: make(map[reflect.Type]typeData),
	}
	ctx.typeQueue = append(ctx.typeQueue, initial...)
	for len(ctx.typeQueue) > 0 {
		car, cdr := ctx.typeQueue[0], ctx.typeQueue[1:]
		ctx.typeQueue = cdr
		if rejected(car, blacklist) {
			continue
		}
		ctx.walkType(car)
	}
	out := make([]typeData, len(ctx.typeData))
	i := 0
	for _, v := range ctx.typeData {
		out[i] = v
		i++
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == out[j].Name {
			return out[i].Package < out[j].Package
		}
		return out[i].Name < out[j].Name
	})
	return out
}
