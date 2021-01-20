package codegen

import (
	"reflect"
	"sort"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/golang/protobuf/protoc-gen-go/generator"
	"github.com/pkg/errors"
	generator2 "github.com/stackrox/rox/central/graphql/generator"
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
	Name        string
	Package     string
	Type        reflect.Type
	FieldData   []fieldData
	UnionData   []unionData
	IsInputType bool
}

type walkState struct {
	typeData  map[reflect.Type]typeData
	typeQueue []typeDescriptor

	skipResolvers []reflect.Type
	skipFields    []generator2.TypeAndField
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
			ctx.typeQueue = append(ctx.typeQueue, typeDescriptor{ty: msgType})
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

type typeDescriptor struct {
	ty          reflect.Type
	isInputType bool
}

func (ctx *walkState) walkType(typeDesc typeDescriptor) {
	var unions []unionData
	ty := typeDesc.ty
	if ty.Kind() == reflect.Slice {
		ty = ty.Elem()
	}
	if ty.Implements(messageType) {
		unions = ctx.walkUnions(ty)
	}
	for ty.Kind() == reflect.Ptr {
		ty = ty.Elem()
	}
	if _, ok := ctx.typeData[ty]; ok {
		return
	}
	td := typeData{Name: ty.Name(), Package: ty.PkgPath(), Type: ty, UnionData: unions}
	for _, union := range unions {
		td.FieldData = append(td.FieldData, union.Entries...)
	}
	if typeDesc.isInputType {
		td.IsInputType = true
	}
	if ty.Kind() == reflect.Struct {
		for i := 0; i < ty.NumField(); i++ {
			ctx.walkField(&td, ty, ty.Field(i))
		}
		sort.Slice(td.FieldData, func(i, j int) bool {
			return td.FieldData[i].Name < td.FieldData[j].Name
		})
	}
	if !rejectedType(ty, ctx.skipResolvers) {
		ctx.typeData[ty] = td
	}
}

func (ctx *walkState) walkField(td *typeData, p reflect.Type, sf reflect.StructField) {
	if len(sf.Name) > 4 && sf.Name[:4] == "XXX_" {
		return
	}
	if strings.HasPrefix(sf.Name, "DEPRECATED") {
		return
	}
	ctx.typeQueue = append(ctx.typeQueue, typeDescriptor{ty: sf.Type})
	if !rejectedField(p, sf, ctx.skipFields) {
		td.FieldData = append(td.FieldData, fieldData{
			Name: sf.Name,
			Type: sf.Type,
		})
	}
}

func rejectedType(p reflect.Type, blacklist []reflect.Type) bool {
	for _, t := range blacklist {
		if t == p {
			return true
		}
	}
	return false
}

func rejectedField(parentType reflect.Type, field reflect.StructField, blacklist []generator2.TypeAndField) bool {
	for _, t := range blacklist {
		if t.ParentType == parentType && t.FieldName == field.Name {
			return true
		}
	}
	return false
}

func typeWalk(parameters generator2.TypeWalkParameters) []typeData {
	ctx := walkState{
		typeData:      make(map[reflect.Type]typeData),
		skipResolvers: parameters.SkipResolvers,
		skipFields:    parameters.SkipFields,
	}
	for _, ty := range parameters.IncludedTypes {
		ctx.typeQueue = append(ctx.typeQueue, typeDescriptor{ty: ty})
	}
	for _, ty := range parameters.InputTypes {
		ctx.typeQueue = append(ctx.typeQueue, typeDescriptor{ty: ty, isInputType: true})
	}
	for len(ctx.typeQueue) > 0 {
		car, cdr := ctx.typeQueue[0], ctx.typeQueue[1:]
		ctx.typeQueue = cdr
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
