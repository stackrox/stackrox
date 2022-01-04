package object

import (
	"reflect"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// StructType distinguishes between different struct types
type StructType int

// Different types of structs
const (
	MESSAGE StructType = iota
	TIME
	ONEOF
)

// Field is the interface that exposes fields in the objects
type Field interface {
	Name() string
	Type() reflect.Kind
	Tags() Tags
}

func newField(field reflect.StructField, kind reflect.Kind) Field {
	return baseField{
		name: field.Name,
		kind: kind,
		tags: Tags{
			StructTag: field.Tag,
		},
	}
}

type baseField struct {
	name string
	kind reflect.Kind
	tags Tags
}

func (f baseField) Name() string {
	return f.name
}

func (f baseField) Type() reflect.Kind {
	return f.kind
}

func (f baseField) Tags() Tags {
	return f.tags
}

// Map defines a map type
type Map struct {
	Field
	Value Field
}

// Slice defines a repeated field
type Slice struct {
	Field
	Value Field
}

// Enum wraps an int32 with the proto descriptor
type Enum struct {
	Field
	Descriptor *descriptor.EnumDescriptorProto
}

// Tags is a utility around the StructTag object
type Tags struct {
	reflect.StructTag
}

// Get returns the tag string for the specified key
func (t Tags) Get(key string) string {
	return t.StructTag.Get(key)
}

// Struct defines a struct in the object
type Struct struct {
	Field
	Fields     []Field
	StructType StructType
}
