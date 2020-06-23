package pathutil

import (
	"reflect"

	"github.com/pkg/errors"
)

// RetrieveValueAtPath takes a path and an object and returns the value found at the path within the object.
func RetrieveValueAtPath(obj interface{}, path *Path) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic while retrieving value: %v", r)
		}
	}()

	currVal := reflect.ValueOf(obj)
	for i, step := range path.steps {
		var err error
		currVal, err = takeStep(currVal, step)
		if err != nil {
			return reflect.Value{}, errors.Errorf("invalid Step %v (idx: %d): %v", step, i, err)
		}
	}

	return currVal.Interface(), nil
}

func takeStep(startVal reflect.Value, step Step) (outputVal reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic executing Step on outputVal %v: %v", startVal, r)
		}
	}()
	switch kind := startVal.Kind(); kind {
	case reflect.Ptr:
		return takeStep(startVal.Elem(), step)
	case reflect.Struct:
		if step.Field() == "" {
			return reflect.Value{}, errors.Errorf("need a field name to traverse struct %s", startVal.Type())
		}
		val := startVal.FieldByName(step.Field())
		if !val.IsValid() {
			return reflect.Value{}, errors.Errorf("struct %s has no field %s", startVal.Type(), step.Field())
		}
		return val, nil
	case reflect.Slice:
		if step.Index() < 0 {
			return reflect.Value{}, errors.Errorf("need to index into slice %v", startVal.Type())
		}
		if length := startVal.Len(); length <= step.Index() {
			return reflect.Value{}, errors.Errorf("index %d greater than length of slice (%d)", step.Index(), length)
		}
		return startVal.Index(step.Index()), nil
	default:
		return reflect.Value{}, errors.Errorf("invalid kind: %v; cannot traverse through it", kind)
	}
}
