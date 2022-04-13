package predicate

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/stackrox/pkg/protoreflect"
	"github.com/stackrox/stackrox/pkg/search"
	"github.com/stackrox/stackrox/pkg/search/predicate/basematchers"
	"github.com/stackrox/stackrox/pkg/stringutils"
	"github.com/stackrox/stackrox/pkg/utils"
)

var (
	timestampPtrType = reflect.TypeOf((*types.Timestamp)(nil))
)

func resultIfNullValue(value string) (*search.Result, bool) {
	if value == "-" {
		return &search.Result{}, true
	}
	return nil, false
}

func formatSingleMatchf(key, template string, val ...interface{}) map[string][]string {
	return map[string][]string{
		key: {fmt.Sprintf(template, val...)},
	}
}

func createBasePredicate(fullPath string, fieldType reflect.Type, value string) (internalPredicate, error) {
	switch fieldType.Kind() {
	case reflect.Ptr:
		return createPtrPredicate(fullPath, fieldType, value)
	case reflect.Array, reflect.Slice:
		return createSlicePredicate(fullPath, fieldType, value)
	case reflect.Map:
		return createMapPredicate(fullPath, fieldType, value)
	case reflect.Bool:
		return createBoolPredicate(fullPath, value)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		if enum, ok := reflect.Zero(fieldType).Interface().(protoreflect.ProtoEnum); ok {
			return createEnumPredicate(fullPath, value, enum)
		}
		return createIntPredicate(fullPath, value)
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		return createUintPredicate(fullPath, value)
	case reflect.Float64, reflect.Float32:
		return createFloatPredicate(fullPath, value)
	case reflect.String:
		return createStringPredicate(fullPath, value)
	}
	return nil, errors.New("unrecognized field")
}

func createPtrPredicate(fullPath string, fieldType reflect.Type, value string) (internalPredicate, error) {
	// Special case for pointer to timestamp.
	if fieldType == timestampPtrType {
		return createTimestampPredicate(fullPath, value)
	}

	// Reroute to element type.
	basePred, err := createBasePredicate(fullPath, fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}

	_, matchOnNull := resultIfNullValue(value)
	if basePred == alwaysTrue && matchOnNull {
		return alwaysTrue, nil
	}
	if basePred == alwaysFalse && !matchOnNull {
		return alwaysFalse, nil
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsZero() || instance.IsNil() {
			return resultIfNullValue(value)
		}
		return basePred.Evaluate(instance.Elem())
	}), nil
}

func createSlicePredicate(fullPath string, fieldType reflect.Type, value string) (internalPredicate, error) {
	basePred, err := createBasePredicate(fullPath, fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsZero() || instance.IsNil() {
			return resultIfNullValue(value)
		}
		for i := 0; i < instance.Len(); i++ {
			if res, match := basePred.Evaluate(instance.Index(i)); match {
				return res, true
			}
		}
		return nil, false
	}), nil
}

func createMapPredicate(fullPath string, fieldType reflect.Type, value string) (internalPredicate, error) {
	key, value := stringutils.Split2(value, "=")

	keyPred, err := createBasePredicate(fullPath, fieldType.Key(), key)
	if err != nil {
		return nil, err
	}
	valPred, err := createBasePredicate(fullPath, fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}

	return createMatchAnyMapPredicate(keyPred, valPred), nil
}

func createMatchAnyMapPredicate(keyPred, valPred internalPredicate) internalPredicate {
	if keyPred == alwaysFalse && valPred == alwaysFalse {
		return alwaysFalse
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsZero() || instance.IsNil() {
			return nil, false
		}

		// The expectation is that we only support searching on map[string]string for now
		iter := instance.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			keyResult, keyMatch := keyPred.Evaluate(key)
			if !keyMatch {
				continue
			}
			valueResult, valueMatch := valPred.Evaluate(val)
			if !valueMatch {
				continue
			}

			return MergeResults(keyResult, valueResult), true
		}
		return nil, false
	})
}

func createBoolPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForBool(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.Kind() != reflect.Bool {
			return nil, false
		}
		instanceAsBool := instance.Bool()
		if baseMatcher(instanceAsBool) {
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, "%t", instanceAsBool),
			}, true
		}
		return nil, false
	}), nil
}

func createEnumPredicate(fullPath, value string, enumRef protoreflect.ProtoEnum) (internalPredicate, error) {
	baseMatcher, numberToName, err := basematchers.ForEnum(value, enumRef)
	if err != nil {
		return nil, err
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			instanceAsInt := instance.Int()
			if baseMatcher(instanceAsInt) {
				matchedValue := numberToName[int32(instanceAsInt)]
				// Should basically never happen.
				if matchedValue == "" {
					utils.Should(errors.Errorf("enum query matched (%s/%s), but no value in numberToName (%d)", fullPath, value, instanceAsInt))
					matchedValue = strconv.Itoa(int(instanceAsInt))
				}
				return &search.Result{
					Matches: formatSingleMatchf(fullPath, matchedValue),
				}, true
			}
		}
		return nil, false
	}), nil
}

func createIntPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForInt(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			instanceAsInt := instance.Int()
			if baseMatcher(instanceAsInt) {
				return &search.Result{
					Matches: formatSingleMatchf(fullPath, "%d", instanceAsInt),
				}, true
			}
		}
		return nil, false
	}), nil
}

func createUintPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForUint(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			instanceAsUInt := instance.Uint()
			if baseMatcher(instanceAsUInt) {
				return &search.Result{
					Matches: formatSingleMatchf(fullPath, "%d", instanceAsUInt),
				}, true
			}
		}
		return nil, false
	}), nil
}

func createFloatPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForFloat(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Float32, reflect.Float64:
			instanceAsFloat := instance.Float()
			if baseMatcher(instanceAsFloat) {
				return &search.Result{
					Matches: formatSingleMatchf(fullPath, "%0.f", instanceAsFloat),
				}, true
			}
		}
		return nil, false
	}), nil
}

func createStringPredicate(fullPath, value string) (internalPredicate, error) {
	baseMatcher, err := basematchers.ForString(value)
	if err != nil {
		return nil, err
	}
	return wrapStringMatcher(fullPath, baseMatcher), nil
}

func wrapStringMatcher(fullPath string, matcher func(string) bool) internalPredicate {
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.Kind() != reflect.String {
			return nil, false
		}
		instanceAsStr := instance.String()
		if matcher(instance.String()) {
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, instanceAsStr),
			}, true
		}
		return nil, false
	})
}
