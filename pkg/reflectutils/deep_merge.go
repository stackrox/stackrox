package reflectutils

import (
	"errors"
	"reflect"

	"github.com/stackrox/rox/pkg/utils"
)

// DeepMergeStructs merges the non-zero structs aStruct and bStruct,
// returning a new struct containing all values from aStruct together with all values from bStruct.
// Values from bStruct take precedence over values from aStruct.
// The type of aStruct (struct vs. struct pointer) pins the type of the return value.
// The returned struct can share references with a and b -- it is NOT a deep copy of the provided
// data structures.
func DeepMergeStructs(aStruct, bStruct interface{}) interface{} {
	aVal := reflect.ValueOf(aStruct)
	bVal := reflect.ValueOf(bStruct)
	utils.Must(assertStruct(aVal))
	utils.Must(assertStruct(bVal))
	return merge(aVal, bVal).Interface()
}

// a and b must be structs of the same type or pointers to structs of the same type.
func merge(a, b reflect.Value) reflect.Value {
	utils.Must(assertTypesEqual(a, b))
	if a.IsZero() {
		return b
	}
	if b.IsZero() {
		return a
	}
	// Both are non-zero.
	returnPointer := false
	if a.Kind() == reflect.Pointer {
		// Reduce layer of indirection.
		returnPointer = true
		a = reflect.Indirect(a)
		b = reflect.Indirect(b)
	}
	newStruct := reflect.New(a.Type()).Elem()
	for fieldNo := 0; fieldNo < a.NumField(); fieldNo++ {
		// Given that the types of A and B are the same, it is safe to assume that the
		// fields of A and B are also the same.
		if !a.Type().Field(fieldNo).IsExported() {
			continue
		}
		fieldA := a.Field(fieldNo)
		fieldB := b.Field(fieldNo)
		newField := newStruct.Field(fieldNo)
		fieldKind := fieldA.Kind()
		switch {
		case fieldKind == reflect.Struct:
			newField.Set(merge(fieldA, fieldB))
		case fieldKind == reflect.Pointer && fieldA.Type().Elem().Kind() == reflect.Struct:
			newField.Set(merge(fieldA, fieldB))
		default:
			if !fieldB.IsZero() {
				newField.Set(fieldB)
			} else {
				newField.Set(fieldA)
			}
		}
	}

	if returnPointer {
		m := reflect.New(newStruct.Type())
		m.Elem().Set(newStruct)
		newStruct = m
	}
	return newStruct
}

func assertStruct(a reflect.Value) error {
	if reflect.Indirect(a).Kind() != reflect.Struct {
		return errors.New("not a struct")
	}
	return nil
}

func assertTypesEqual(a reflect.Value, b reflect.Value) error {
	aType := a.Type()
	bType := b.Type()
	if aType != bType {
		return errors.New("type mismatch")
	}
	return nil
}
