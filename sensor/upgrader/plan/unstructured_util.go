package plan

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func nestedValueNoCopyOrError[T any](obj map[string]interface{}, fields ...string) (T, error) {
	var val T
	valRaw, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if err != nil {
		return val, errors.Wrapf(err, "retrieving field %s", strings.Join(fields, "."))
	}
	if !found {
		return val, errors.Errorf("field %s not found", strings.Join(fields, "."))
	}
	val, ok := valRaw.(T)
	if !ok {
		return val, errors.Errorf("field %s is of unexpected type %T, not %T", strings.Join(fields, "."), valRaw, val)
	}
	return val, nil
}

func nestedNonZeroValueNoCopyOrError[T comparable](obj map[string]interface{}, fields ...string) (T, error) {
	var zeroVal T
	val, err := nestedValueNoCopyOrError[T](obj, fields...)
	if err != nil {
		return val, err
	}
	if val == zeroVal {
		return val, errors.Errorf("field %s has the zero value", strings.Join(fields, "."))
	}
	return val, nil
}

func nestedValueNoCopyOrDefault[T any](obj map[string]interface{}, def T, fields ...string) T {
	valRaw, found, err := unstructured.NestedFieldNoCopy(obj, fields...)
	if err != nil || !found {
		return def
	}
	val, ok := valRaw.(T)
	if !ok {
		return def
	}
	return val
}
