package sac

import (
	"context"
	"reflect"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/reflectutils"
)

type objPredPair struct {
	obj  interface{}
	pred ScopePredicate
}

// ObjectFilter allows efficiently filtering (wrt. SAC constraints) objects.
type ObjectFilter struct {
	checker ScopeChecker
	allowed []interface{}
	maybe   []objPredPair
}

// NewObjectFilter creates a new object filter instance.
func NewObjectFilter(checker ScopeChecker) *ObjectFilter {
	return &ObjectFilter{
		checker: checker,
	}
}

// Add adds an object to the filter, using the given predicate to determine whether it is allowed.
func (f *ObjectFilter) Add(obj interface{}, pred ScopePredicate) {
	if res := pred.TryAllowed(f.checker); res == Allow {
		f.allowed = append(f.allowed, obj)
	} else if res == Unknown {
		f.maybe = append(f.maybe, objPredPair{
			obj:  obj,
			pred: pred,
		})
	}
}

// GetAllowed returns the list of allowed objects, or an error.
func (f *ObjectFilter) GetAllowed(ctx context.Context) ([]interface{}, error) {
	currMaybe := f.maybe
	f.maybe = nil

	if len(currMaybe) > 0 {
		if err := f.checker.PerformChecks(ctx); err != nil {
			return nil, err
		}
		for _, objAndPred := range currMaybe {
			f.Add(objAndPred.obj, objAndPred.pred)
		}

		if len(f.maybe) > 0 {
			return nil, errors.New("still Unknown objects after second iteration")
		}
	}

	allowed := f.allowed
	f.allowed = nil

	return allowed, nil
}

// FilterSlice filters the given slice of objects, using scopePredFunc to determine the scope predicate for each object.
func FilterSlice(ctx context.Context, sc ScopeChecker, objs []interface{}, scopePredFunc func(interface{}) ScopePredicate) ([]interface{}, error) {
	if ok, err := sc.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
		return objs, nil
	}

	f := NewObjectFilter(sc)
	for i := range objs {
		obj := objs[i]
		f.Add(obj, scopePredFunc(obj))
	}
	return f.GetAllowed(ctx)
}

var (
	scopePredTy = reflect.TypeOf((*ScopePredicate)(nil)).Elem()
)

// FilterSliceReflect uses reflection to filter the given typed slice, applying a typed predicate function to obtain
// scope keys.
func FilterSliceReflect(ctx context.Context, sc ScopeChecker, objSlice interface{}, scopePredFunc interface{}) (interface{}, error) {
	if ok, err := sc.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
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
	if ok, err := sc.Allowed(ctx); err != nil {
		return nil, err
	} else if ok {
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
