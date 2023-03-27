package testutils

import (
	"errors"
	"math/rand"
	"reflect"
	"strings"
	"time"
	"unicode"
	"unsafe"

	"github.com/stackrox/rox/pkg/uuid"
)

// BasicTypeInitializer prescribes how to initialize a struct field with a given type.
type BasicTypeInitializer interface {
	Value(ty reflect.Type, fieldPath []reflect.StructField) interface{}
}

// UniqueTypeInitializer prescribes how to initialize a struct field with a given type.
type UniqueTypeInitializer interface {
	ValueUnique(ty reflect.Type, fieldPath []reflect.StructField) interface{}
}

type zeroInitializer struct{}

func (zeroInitializer) Value(ty reflect.Type, _ []reflect.StructField) interface{} {
	return reflect.Zero(ty).Interface()
}

// ZeroInitializer returns a BasicTypeInitializer that initializes all fields of basic types with their zero value
func ZeroInitializer() BasicTypeInitializer {
	return zeroInitializer{}
}

type simpleInitializer struct{}

func (simpleInitializer) Value(ty reflect.Type, _ []reflect.StructField) interface{} {
	switch ty.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 1
	case reflect.Float32, reflect.Float64:
		return 1.0
	case reflect.Complex64, reflect.Complex128:
		return 1.0i
	case reflect.Bool:
		return true
	case reflect.String:
		return uuid.NewDummy().String()
	}
	return nil
}

type uniqueInitializer struct{}

func (uniqueInitializer) Value(ty reflect.Type, _ []reflect.StructField) interface{} {
	// seed rand
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	switch ty.Kind() {
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return r.Int31()
	case reflect.Int8, reflect.Uint8:
		// We are using Uint8 for bytes that become varchars.  Need to ensure that we return a
		// non-zero number within the Uint8 range of values.
		return r.Intn(100) + 1
	case reflect.Float32, reflect.Float64:
		return r.Float32()
	case reflect.Complex64, reflect.Complex128:
		return complex(r.Float32(), 1.0)
	case reflect.Bool:
		return true
	case reflect.String:
		return uuid.NewV4().String()
	}
	return nil
}

// SimpleInitializer returns a BasicTypeInitializer that initializes all fields of basic types with a simple non-zero
// value (1 for integer fields, 1.0 for float fields, true for boolean fields, "a" for string fields).
func SimpleInitializer() BasicTypeInitializer {
	return simpleInitializer{}
}

// UniqueInitializer returns a UniqueTypeInitializer that initializes all fields of basic types with a simple non-zero
// value (1 for integer fields, 1.0 for float fields, true for boolean fields, a new UUID for string fields).
func UniqueInitializer() BasicTypeInitializer {
	return uniqueInitializer{}
}

// FieldFilter determines whether or not to include a field.
type FieldFilter func(field reflect.StructField, ancestors []reflect.StructField) bool

// JSONFieldsFilter is a field filter that includes only JSON fields.
func JSONFieldsFilter(field reflect.StructField, _ []reflect.StructField) bool {
	if field.Name != "" && unicode.IsLower([]rune(field.Name)[0]) {
		return false
	}
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" {
		parts := strings.SplitN(jsonTag, ",", 2)
		if parts[0] == "-" {
			return false
		}
	}
	return true
}

// FullInit fully initializes the given value, making sure all fields are set. This can be used, e.g., to make sure that
// all fields are written in serialized outputs.
// Pointers are initialized with newly created elements of pointee types with are initialized recursively. Slices and
// maps are initialized with one-element instances with recursively initialized elements/key-value pairs. For arrays,
// all elements are initialized recursively. Basic types are initialized according to the given BasicTypeInitializer.
// The field filter, if non-nil, causes pruning at selected (sub-)fields.
func FullInit(val interface{}, init BasicTypeInitializer, fieldFilter FieldFilter) error {
	rval := reflect.ValueOf(val)
	if rval.Kind() != reflect.Ptr || rval.IsNil() {
		return errors.New("argument to FullInit must be a non-nil pointer")
	}
	fullInitRecursive(rval.Elem(), init, fieldFilter, nil, make(map[reflect.Type]struct{}))
	return nil
}

func fullInitRecursive(val reflect.Value, init BasicTypeInitializer, fieldFilter FieldFilter, fieldPath []reflect.StructField, seenTypes map[reflect.Type]struct{}) {
	switch val.Kind() {
	case reflect.Func, reflect.Interface, reflect.Uintptr, reflect.UnsafePointer, reflect.Invalid:
		// nothing we can do here

	case reflect.Ptr:
		if _, ok := seenTypes[val.Type().Elem()]; !ok {
			val.Set(reflect.New(val.Type().Elem()))
			fullInitRecursive(val.Elem(), init, fieldFilter, fieldPath, seenTypes)
		}

	case reflect.Slice:
		val.Set(reflect.MakeSlice(val.Type(), 1, 1))
		fullInitRecursive(val.Index(0), init, fieldFilter, fieldPath, seenTypes)

	case reflect.Array:
		elemVal := reflect.New(val.Type().Elem()).Elem()
		fullInitRecursive(elemVal, init, fieldFilter, fieldPath, seenTypes)
		for i := 0; i < val.Type().Len(); i++ {
			val.Index(i).Set(elemVal)
		}

	case reflect.Chan:
		val.Set(reflect.MakeChan(val.Type(), 0))

	case reflect.Map:
		val.Set(reflect.MakeMap(val.Type()))
		keyVal := reflect.New(val.Type().Key()).Elem()
		fullInitRecursive(keyVal, init, fieldFilter, fieldPath, seenTypes)
		valueVal := reflect.New(val.Type().Elem()).Elem()
		fullInitRecursive(valueVal, init, fieldFilter, fieldPath, seenTypes)
		val.SetMapIndex(keyVal, valueVal)

	case reflect.Struct:
		fullInitStruct(val, init, fieldFilter, fieldPath, seenTypes)

	default:
		val.Set(reflect.ValueOf(init.Value(val.Type(), fieldPath)).Convert(val.Type()))
	}
}

func fullInitStruct(structVal reflect.Value, init BasicTypeInitializer, fieldFilter FieldFilter, fieldPath []reflect.StructField, seenTypes map[reflect.Type]struct{}) {
	structTy := structVal.Type()
	seenTypes[structTy] = struct{}{}
	defer delete(seenTypes, structTy)

	for i := 0; i < structTy.NumField(); i++ {
		field := structTy.Field(i)
		if fieldFilter != nil {
			if !fieldFilter(field, fieldPath) {
				continue
			}
		}

		fieldVal := structVal.FieldByIndex(field.Index)
		if field.Name != "" && unicode.IsLower([]rune(field.Name)[0]) {
			// If a field is not exported, we need to make it writable with the following hack.
			//#nosec G103
			fieldVal = reflect.NewAt(fieldVal.Type(), unsafe.Pointer(fieldVal.UnsafeAddr())).Elem()
		}
		fullInitRecursive(fieldVal, init, fieldFilter, append(fieldPath, field), seenTypes)
	}
}
