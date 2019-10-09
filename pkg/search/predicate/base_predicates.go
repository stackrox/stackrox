package predicate

import (
	"errors"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/pkg/search"
)

var (
	timestampPtrType = reflect.TypeOf((*types.Timestamp)(nil))
)

func createBasePredicate(fieldType reflect.Type, value string) (internalPredicate, error) {
	switch fieldType.Kind() {
	case reflect.Ptr:
		return createPtrPredicate(fieldType, value)
	case reflect.Array:
		return createSlicePredicate(fieldType, value)
	case reflect.Slice:
		return createSlicePredicate(fieldType, value)
	case reflect.Map:
		return createMapPredicate(fieldType, value)
	case reflect.Bool:
		return createBoolPredicate(value)
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		return createIntPredicate(value)
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		return createUintPredicate(value)
	case reflect.Float64, reflect.Float32:
		return createFloatPredicate(value)
	case reflect.String:
		return createStringPredicate(value)
	}
	return nil, errors.New("unrecognized field")
}

func createPtrPredicate(fieldType reflect.Type, value string) (internalPredicate, error) {
	// Special case for pointer to timestamp.
	if fieldType == timestampPtrType {
		return createTimestampPredicate(value)
	}

	// Reroute to element type.
	basePred, err := createBasePredicate(fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		if instance.IsZero() || instance.IsNil() {
			return false
		}
		return basePred(instance.Elem())
	}, nil
}

func createSlicePredicate(fieldType reflect.Type, value string) (internalPredicate, error) {
	basePred, err := createBasePredicate(fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}

	return func(instance reflect.Value) bool {
		if instance.IsZero() || instance.IsNil() {
			return false
		}
		for i := 0; i < instance.Len(); i++ {
			if basePred(instance.Index(i)) {
				return true
			}
		}
		return false
	}, nil
}

func createMapPredicate(fieldType reflect.Type, value string) (internalPredicate, error) {
	keyPred, err := createBasePredicate(fieldType.Key(), value)
	if err != nil {
		return nil, err
	}
	valPred, err := createBasePredicate(fieldType.Elem(), value)
	if err != nil {
		return nil, err
	}

	return func(instance reflect.Value) bool {
		if instance.IsZero() || instance.IsNil() {
			return false
		}
		iter := instance.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			if keyPred(key) || valPred(val) {
				return true
			}
		}
		return false
	}, nil
}

func createBoolPredicate(value string) (internalPredicate, error) {
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		if instance.Kind() != reflect.Bool {
			return false
		}
		return instance.Bool() == boolValue
	}, nil
}

func createIntPredicate(value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := intComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	intValue, err := parseInt(value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			return comparator(instance.Int(), intValue)
		}
		return false
	}, nil
}

func createUintPredicate(value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := uintComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	uintValue, err := parseUint(value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		switch instance.Kind() {
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			return comparator(instance.Uint(), uintValue)
		}
		return false
	}, nil
}

func createFloatPredicate(value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := floatComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	floatValue, err := parseFloat(value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		switch instance.Kind() {
		case reflect.Float32, reflect.Float64:
			return comparator(instance.Float(), floatValue)
		}
		return false
	}, nil
}

func createTimestampPredicate(value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := timestampComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	timestampValue, err := parseTimestamp(value)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) bool {
		return comparator(instance.Interface(), timestampValue)
	}, nil
}

func createStringPredicate(value string) (internalPredicate, error) {
	if strings.HasPrefix(value, search.RegexPrefix) {
		value = strings.TrimPrefix(value, search.RegexPrefix)
		return stringRegexPredicate(value)
	} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return stringExactPredicate(value[1 : len(value)-1])
	}
	return stringPrefixPredicate(value)
}

func stringRegexPredicate(value string) (func(value reflect.Value) bool, error) {
	matcher, err := regexp.Compile(value)
	if err != nil {
		return nil, err
	}
	return wrapStringPredicate(func(instance string) bool {
		return matcher.MatchString(instance)
	}), nil
}

func stringExactPredicate(value string) (func(value reflect.Value) bool, error) {
	return wrapStringPredicate(func(instance string) bool {
		return instance == value
	}), nil
}

func stringPrefixPredicate(value string) (internalPredicate, error) {
	return wrapStringPredicate(func(instance string) bool {
		return strings.HasPrefix(instance, value)
	}), nil
}

func wrapStringPredicate(pred func(string) bool) internalPredicate {
	return func(instance reflect.Value) bool {
		if instance.Kind() != reflect.String {
			return false
		}
		return pred(instance.String())
	}
}
