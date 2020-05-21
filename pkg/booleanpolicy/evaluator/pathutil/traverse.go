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
		currVal, err = takeStep(currVal, &path.steps[i])
		if err != nil {
			return reflect.Value{}, errors.Errorf("invalid step %v (idx: %d): %v", step, i, err)
		}
	}

	return currVal.Interface(), nil
}

func takeStep(startVal reflect.Value, step *step) (outputVal reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("panic executing step on outputVal %v: %v", startVal, r)
		}
	}()
	switch kind := startVal.Kind(); kind {
	case reflect.Ptr:
		return takeStep(startVal.Elem(), step)
	case reflect.Struct:
		if step.field == "" {
			return reflect.Value{}, errors.Errorf("need a field name to traverse struct %s", startVal.Type())
		}
		val := startVal.FieldByName(step.field)
		if !val.IsValid() {
			return reflect.Value{}, errors.Errorf("struct %s has no field %s", startVal.Type(), step.field)
		}
		return val, nil
	case reflect.Slice:
		if step.index == nil {
			return reflect.Value{}, errors.Errorf("need to index into slice %v", startVal.Type())
		}
		if length := startVal.Len(); length <= *step.index {
			return reflect.Value{}, errors.Errorf("index %d greater than length of slice (%d)", *step.index, length)
		}
		return startVal.Index(*step.index), nil
	default:
		return reflect.Value{}, errors.Errorf("invalid kind: %v; cannot traverse through it", kind)
	}
}
