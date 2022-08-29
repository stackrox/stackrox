package protowalk

import (
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/stringutils"
)

// FieldKind represents the kind of a field.
type FieldKind int

const (
	// MessageFieldKind indicates that the element of the respective field is of message type (struct pointer), and
	// thus can have zero or more child fields.
	MessageFieldKind FieldKind = iota
	// OneofFieldKind indicates that the element of the repsective field is a oneof wrapper (interface that is nil
	// or a struct wrapper), and has exactly one child field.
	OneofFieldKind
	// LeafFieldKind indicates that the element of the respective field is neither a message nor oneof, and does not
	// have any child fields.
	LeafFieldKind
)

// Field represents a field in a proto message type or oneof wrapper.
type Field struct {
	// ContainingType is a pointer to the message or oneof wrapper type to which this field belongs.
	// It is always a struct pointer.
	ContainingType reflect.Type
	// StructField is the definition of the struct field within the containing type.
	reflect.StructField
}

// ElemType is the type of element stored in this field. In case of slices (repeated fields), this is the slice
// element type; in case of maps, this is the value type. Otherwise, it is the type of the respective struct field.
func (f Field) ElemType() reflect.Type {
	ty := f.Type
	if ty.Kind() == reflect.Slice || ty.Kind() == reflect.Map {
		ty = ty.Elem()
	}
	return ty
}

// Kind returns the kind of this field.
func (f Field) Kind() FieldKind {
	elemTy := f.ElemType()
	if elemTy.Kind() == reflect.Interface {
		return OneofFieldKind
	}
	if elemTy.Kind() == reflect.Ptr && elemTy.Elem().Kind() == reflect.Struct {
		return MessageFieldKind
	}
	return LeafFieldKind
}

// IsLeaf is a shorthand for f.Kind() == LeafFieldKind.
func (f Field) IsLeaf() bool {
	return f.Kind() == LeafFieldKind
}

// ProtoName returns the name of the field at the protobuf level. For oneof fields, it will return the empty string.
func (f Field) ProtoName() string {
	return protoTagValue(f.Tag.Get("protobuf"), "name")
}

// JSONPBName returns the name of the field as when serialized with JSONPb. For oneof fields, it will return the empty
// string.
func (f Field) JSONPBName() string {
	return protoTagValue(f.Tag.Get("protobuf"), "json")
}

// JSONName returns the name of the field as when serialized via json.Marshal. For oneof fields, it will return the
// name of the Go field.
func (f Field) JSONName() string {
	jsonTag := f.Tag.Get("json")
	if jsonTag == "" {
		return f.Name
	}
	name, _ := stringutils.Split2(jsonTag, ",")
	return name
}

// FieldPath is a sequence of Fields, indicating some field in a proto message accessible from a top-level object.
// A FieldPath p must always satisfy the following:
//   - for 0 <= i < len(p) - 1, p[i].IsLeaf() must return false
//   - for 0 <= i < len(p) - 1, p[i].ElemType() must either be the same type as p[i+1].ContainingType, or an interface
//     implemented by p[i+1].ContainingType.
type FieldPath []Field

// ProtoPathElems returns the path elements of the protobuf field path to this field.
func (p FieldPath) ProtoPathElems() []string {
	result := make([]string, 0, len(p))
	for _, f := range p {
		fname := f.ProtoName()
		if fname == "" {
			continue
		}
		result = append(result, fname)
	}
	return result
}

// ProtoPath returns the protobuf field path representation, i.e., the protobuf names of all fields separated
// by dots.
func (p FieldPath) ProtoPath() string {
	return strings.Join(p.ProtoPathElems(), ".")
}

func (p FieldPath) JSONPathElems() []string {
	result := make([]string, 0, len(p))
	for _, f := range p {
		fname := f.JSONName()
		if fname == "" {
			continue
		}
		result = append(result, fname)
	}
	return result
}

func (p FieldPath) JSONPath() string {
	return strings.Join(p.JSONPathElems(), ".")
}

// StructFields returns the sequence of reflect.StructFields corresponding to this path.
func (p FieldPath) StructFields() []reflect.StructField {
	sfp := make([]reflect.StructField, 0, len(p))
	for _, elem := range p {
		sfp = append(sfp, elem.StructField)
	}
	return sfp
}

// Field returns the Field that this path refers to, i.e., the last field in the path. This method is provided for
// convenience only.
func (p FieldPath) Field() Field {
	return p[len(p)-1]
}

func protoTagValue(protoTag string, key string) string {
	elems := strings.Split(protoTag, ",")
	keyPrefix := key + "="
	for _, e := range elems {
		if stringutils.ConsumePrefix(&e, keyPrefix) {
			return e
		}
	}
	return ""
}

func fieldName(protoTag string) string {
	elems := strings.Split(protoTag, ",")
	for _, e := range elems {
		if stringutils.ConsumePrefix(&e, "name=") {
			return e
		}
	}
	return ""
}
