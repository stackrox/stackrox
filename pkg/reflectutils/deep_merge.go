package reflectutils

import (
	"errors"
	"reflect"

	"github.com/stackrox/rox/pkg/utils"
)

// DeepMergeStructs merges the non-zero structs aStruct and bStruct,
// returning a new struct containing all values from aStruct together with all values from bStruct.
// Values from bStruct take precedence over values from aStruct.
// The type of aStruct (struct vs. struct pointer) selects the type of the return value.
func DeepMergeStructs(aStruct, bStruct interface{}) (merged interface{}) {
	var mergedVal reflect.Value
	aVal := reflect.ValueOf(aStruct)
	if aVal.Kind() == reflect.Ptr {
		defer func() {
			ptr := reflect.New(mergedVal.Type())
			ptr.Elem().Set(mergedVal)
			merged = ptr.Interface()
		}()
	}
	utils.Must(assertStruct(aVal))
	bVal := reflect.ValueOf(bStruct)
	utils.Must(assertStruct(bVal))
	mergedVal = merge(aVal, bVal)
	merged = mergedVal.Interface()
	return
}

// a and b must be structs.
func merge(a, b reflect.Value) reflect.Value {
	a = reflect.Indirect(a)
	b = reflect.Indirect(b)
	utils.Must(assertTypesEqual(a, b))
	merged := reflect.New(a.Type()).Elem()
	for fieldNo := 0; fieldNo < a.NumField(); fieldNo++ {
		if !a.Type().Field(fieldNo).IsExported() {
			continue
		}
		fieldA := a.Field(fieldNo)
		fieldB := b.Field(fieldNo)
		fieldMerged := merged.Field(fieldNo)
		switch fieldA.Kind() {
		case reflect.Struct:
			fieldMerged.Set(merge(fieldA, fieldB))
		default:
			fieldMerged.Set(fieldA)
			if !fieldB.IsZero() {
				fieldMerged.Set(fieldB)
			}
		}
	}
	return merged
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
