package lint

import (
	"reflect"
	"strings"

	"github.com/stackrox/rox/pkg/stringutils"
)

type ProtoFieldKind int

const (
	MessageFieldKind ProtoFieldKind = iota
	OneofFieldKind
	LeafFieldKind
)

type ProtoField struct {
	// ContainingType is a pointer to the message or oneof wrapper type to which this field belongs.
	// It is always a struct pointer.
	ContainingType reflect.Type
	// StructField is the definition of the struct field within the containing type.
	reflect.StructField
}

// ElemType is the type of element stored in this field. In case of slices (repeated fields), this is the slice
// element type; in case of maps, this is the value type. Otherwise, it is the type of the respective struct field.
func (f ProtoField) ElemType() reflect.Type {
	ty := f.Type
	if ty.Kind() == reflect.Slice || ty.Kind() == reflect.Map {
		ty = ty.Elem()
	}
	return ty
}

func (f ProtoField) Kind() ProtoFieldKind {
	elemTy := f.ElemType()
	if elemTy.Kind() == reflect.Interface {
		return OneofFieldKind
	}
	if elemTy.Kind() == reflect.Ptr && elemTy.Elem().Kind() == reflect.Struct {
		return MessageFieldKind
	}
	return LeafFieldKind
}

func (f ProtoField) IsLeaf() bool {
	return f.Kind() == LeafFieldKind
}

func (f ProtoField) ProtoName() string {
	return fieldName(f.Tag.Get("protobuf"))
}

type ProtoFieldPath []ProtoField

func (p ProtoFieldPath) ProtoPathElems() []string {
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

func (p ProtoFieldPath) ProtoPath() string {
	return strings.Join(p.ProtoPathElems(), ".")
}

func (p ProtoFieldPath) Field() ProtoField {
	return p[len(p)-1]
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

type WalkCB = func(fieldPath ProtoFieldPath) bool

func WalkProto(ty reflect.Type, fieldPath ProtoFieldPath, walkCB WalkCB) {
	if len(fieldPath) > 0 {
		if !walkCB(fieldPath) {
			return
		}
	}

	if ty.Kind() == reflect.Interface {
		oneofWrappers := reflect.Zero(fieldPath.Field().ContainingType).Interface().(interface{ XXX_OneofWrappers() []interface{} }).XXX_OneofWrappers()
		for _, w := range oneofWrappers {
			wrapperTy := reflect.TypeOf(w)
			if !wrapperTy.Implements(ty) {
				continue
			}
			WalkProto(wrapperTy, fieldPath, walkCB)
		}
		return
	}

	if ty.Kind() != reflect.Ptr || ty.Elem().Kind() != reflect.Struct {
		return
	}

	elemTy := ty.Elem()
	for i := 0; i < elemTy.NumField(); i++ {
		f := elemTy.Field(i)

		if f.Tag.Get("protobuf") == "" && f.Type.Kind() != reflect.Interface {
			continue // not a proto or oneof wrapper field
		}

		nextPath := append(fieldPath, ProtoField{ContainingType: ty, StructField: f})
		WalkProto(nextPath.Field().ElemType(), nextPath, walkCB)
	}
}
