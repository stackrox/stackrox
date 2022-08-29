package sac

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/reflectutils"
)

// ObjectFilter allows efficiently filtering (wrt. SAC constraints) objects.
type ObjectFilter struct {
	checker ScopeChecker
	allowed []interface{}
}

// NewObjectFilter creates a new object filter instance.
func NewObjectFilter(checker ScopeChecker) *ObjectFilter {
	return &ObjectFilter{
		checker: checker,
	}
}

// Add adds an object to the filter, using the given predicate to determine whether it is allowed.
func (f *ObjectFilter) Add(obj interface{}, pred ScopePredicate) {
	if pred.Allowed(f.checker) {
		f.allowed = append(f.allowed, obj)
	}
}

// GetAllowed returns the list of allowed objects, or an error.
func (f *ObjectFilter) GetAllowed(ctx context.Context) ([]interface{}, error) {
	allowed := f.allowed
	f.allowed = nil

	return allowed, nil
}

var (
	scopePredTy = reflect.TypeOf((*ScopePredicate)(nil)).Elem()
)

// FilterSliceReflect uses reflection to filter the given typed slice, applying a typed predicate function to obtain
// scope keys.
func FilterSliceReflect(ctx context.Context, sc ScopeChecker, objSlice interface{}, scopePredFunc interface{}) (interface{}, error) {
	if sc.IsAllowed() {
		return objSlice, nil
	}

	sliceVal := reflect.ValueOf(objSlice)
	if sliceVal.Kind() != reflect.Slice {
		return nil, errors.New("argument is not a slice")
	}
	scopePredFuncVal := reflect.ValueOf(scopePredFunc)
	if scopePredFuncVal.Kind() != reflect.Func {
		return nil, errors.New("argument is not a function")
	}
	if scopePredFuncVal.Type().NumIn() != 1 || scopePredFuncVal.Type().NumOut() != 1 {
		return nil, errors.New("predicate function has wrong signature")
	}

	f := NewObjectFilter(sc)
	sliceLen := sliceVal.Len()
	for i := 0; i < sliceLen; i++ {
		objVal := sliceVal.Index(i)
		predVal := scopePredFuncVal.Call([]reflect.Value{objVal})

		f.Add(objVal.Interface(), predVal[0].Convert(scopePredTy).Interface().(ScopePredicate))
	}

	allowed, err := f.GetAllowed(ctx)
	if err != nil {
		return nil, err
	}
	return reflectutils.ToTypedSlice(allowed, sliceVal.Type().Elem()), nil
}

// FilterMapReflect uses reflection to filter the given typed map, applying a typed predicate function to obtain
// scope keys.
// If the scopePredFunc takes in two arguments, the arguments are the key and the value. Otherwise, just the value is
// passed.
func FilterMapReflect(ctx context.Context, sc ScopeChecker, objMap interface{}, scopePredFunc interface{}) (interface{}, error) {
	if sc.IsAllowed() {
		return objMap, nil
	}

	mapVal := reflect.ValueOf(objMap)
	if mapVal.Kind() != reflect.Map {
		return nil, errors.New("argument is not a map")
	}
	scopePredFuncVal := reflect.ValueOf(scopePredFunc)
	if scopePredFuncVal.Kind() != reflect.Func {
		return nil, errors.New("argument is not a function")
	}
	numIn := scopePredFuncVal.Type().NumIn()
	if numIn < 1 || numIn > 2 || scopePredFuncVal.Type().NumOut() != 1 {
		return nil, errors.New("predicate function has wrong signature")
	}

	f := NewObjectFilter(sc)
	var args [2]reflect.Value
	for iter := mapVal.MapRange(); iter.Next(); {
		keyVal := iter.Key()
		valueVal := iter.Value()
		if numIn == 2 {
			args[0] = keyVal
		}
		args[numIn-1] = valueVal

		predVal := scopePredFuncVal.Call(args[:numIn])
		f.Add(keyVal, predVal[0].Convert(scopePredTy).Interface().(ScopePredicate))
	}

	allowed, err := f.GetAllowed(ctx)
	if err != nil {
		return nil, err
	}

	outMap := reflect.MakeMapWithSize(reflect.MapOf(mapVal.Type().Key(), mapVal.Type().Elem()), len(allowed))
	for _, key := range allowed {
		keyVal := key.(reflect.Value)
		outMap.SetMapIndex(keyVal, mapVal.MapIndex(keyVal))
	}
	return outMap.Interface(), nil
}
