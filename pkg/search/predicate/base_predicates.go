package predicate

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/regexutils"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/stringutils"
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

	// This is a hack! It relies on all "required label" policies using negated queries and all "disallowed label"
	// policies using non-negated queries.  It should definitely change after we get rid of search based policies.
	matchAll := strings.HasPrefix(key, search.NegationPrefix) || strings.HasPrefix(value, search.NegationPrefix)

	if matchAll {
		return createMatchAllMapPredicate(keyPred, valPred), nil
	}
	return createMatchAnyMapPredicate(keyPred, valPred), nil
}

func createMatchAllMapPredicate(keyPred, valPred internalPredicate) internalPredicate {
	if keyPred == alwaysTrue && valPred == alwaysTrue {
		return alwaysTrue
	}

	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.IsZero() || instance.IsNil() {
			// This is a hack!  This path is used by RequiredMapValue policies so it needs to return true if the
			// required value isn't in the map even though there were no matches in the empty map.  This should
			// definitely change after we get rid of search based policies.
			return &search.Result{}, true
		}

		// The expectation is that we only support searching on map[string]string for now
		iter := instance.MapRange()
		for iter.Next() {
			key := iter.Key()
			val := iter.Value()
			_, keyMatch := keyPred.Evaluate(key)
			if !keyMatch {
				return nil, false
			}
			_, valueMatch := valPred.Evaluate(val)
			if !valueMatch {
				return nil, false
			}
		}
		return &search.Result{}, true
	})
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
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.Kind() != reflect.Bool {
			return nil, false
		}
		if instance.Bool() != boolValue {
			return nil, false
		}
		return &search.Result{
			Matches: formatSingleMatchf(fullPath, "%t", instance.Bool()),
		}, true
	}), nil
}

func createEnumPredicate(fullPath, value string, enumRef protoreflect.ProtoEnum) (internalPredicate, error) {
	// Map the enum strings to integer values.
	enumDesc, err := protoreflect.GetEnumDescriptor(enumRef)
	if err != nil {
		return nil, err
	}
	nameToNumber := mapEnumValues(enumDesc)

	// Get the comparator if needed.
	cmpStr, value := getNumericComparator(value)

	// Translate input value to an int if needed.
	var int64Value int64
	int32Value, hasIntValue := nameToNumber[strings.ToLower(value)]
	if hasIntValue {
		int64Value = int64(int32Value)
	} else {
		return nil, errors.Errorf("unrecognized enum value: %s in %+v", value, nameToNumber)
	}

	// Generate the comparator for the integer values.
	comparator, err := intComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			if comparator(instance.Int(), int64Value) {
				return &search.Result{
					Matches: formatSingleMatchf(fullPath, "%d", instance.Int()),
				}, true
			}
		}
		return nil, false
	}), nil
}

func createIntPredicate(fullPath, value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := intComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	intValue, err := parseInt(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			if !comparator(instance.Int(), intValue) {
				return nil, false
			}
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, "%d", instance.Int()),
			}, true
		}
		return nil, false
	}), nil
}

func createUintPredicate(fullPath, value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := uintComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	uintValue, err := parseUint(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			if !comparator(instance.Uint(), uintValue) {
				return nil, false
			}
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, "%d", instance.Uint()),
			}, true
		}
		return nil, false
	}), nil
}

func createFloatPredicate(fullPath, value string) (internalPredicate, error) {
	cmpStr, value := getNumericComparator(value)
	comparator, err := floatComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	floatValue, err := parseFloat(value)
	if err != nil {
		return nil, err
	}
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		switch instance.Kind() {
		case reflect.Float32, reflect.Float64:
			if !comparator(instance.Float(), floatValue) {
				return nil, false
			}
			return &search.Result{
				Matches: formatSingleMatchf(fullPath, "%0.f", instance.Float()),
			}, true
		}
		return nil, false
	}), nil
}

func createStringPredicate(fullPath, value string) (internalPredicate, error) {
	negated := strings.HasPrefix(value, search.NegationPrefix)
	if negated {
		value = strings.TrimPrefix(value, search.NegationPrefix)
	}
	if strings.HasPrefix(value, search.RegexPrefix) {
		value = strings.TrimPrefix(value, search.RegexPrefix)
		return stringRegexPredicate(fullPath, value, negated)
	} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) > 1 {
		return stringExactPredicate(fullPath, value[1:len(value)-1], negated)
	}
	return stringPrefixPredicate(fullPath, value, negated)
}

func stringRegexPredicate(fullPath, value string, negated bool) (internalPredicate, error) {
	matcher, err := regexp.Compile(value)
	if err != nil {
		return nil, err
	}
	return wrapStringPredicate(func(instance string) (*search.Result, bool) {
		// matched == negated is equivalent to !(matched XOR negated), which is what we want here
		if regexutils.MatchWholeString(matcher, instance) == negated {
			return nil, false
		}

		return &search.Result{
			Matches: formatSingleMatchf(fullPath, instance),
		}, true
	}), nil
}

func stringExactPredicate(fullPath, value string, negated bool) (internalPredicate, error) {
	return wrapStringPredicate(func(instance string) (*search.Result, bool) {
		// matched == negated is equivalent to !(matched XOR negated), which is what we want here
		if (instance == value) == negated {
			return nil, false
		}
		return &search.Result{
			Matches: formatSingleMatchf(fullPath, instance),
		}, true
	}), nil
}

func stringPrefixPredicate(fullPath, value string, negated bool) (internalPredicate, error) {
	return wrapStringPredicate(func(instance string) (*search.Result, bool) {
		// matched == negated is equivalent to !(matched XOR negated), which is what we want here
		if (value == search.WildcardString || strings.HasPrefix(instance, value)) == negated {
			return nil, false
		}
		return &search.Result{
			Matches: formatSingleMatchf(fullPath, instance),
		}, true
	}), nil
}

func wrapStringPredicate(pred func(string) (*search.Result, bool)) internalPredicate {
	return internalPredicateFunc(func(instance reflect.Value) (*search.Result, bool) {
		if instance.Kind() != reflect.String {
			return nil, false
		}
		return pred(instance.String())
	})
}

func mapEnumValues(enumDesc *descriptor.EnumDescriptorProto) (nameToNumber map[string]int32) {
	nameToNumber = make(map[string]int32)
	for _, v := range enumDesc.GetValue() {
		lName := strings.ToLower(v.GetName())
		nameToNumber[lName] = v.GetNumber()
	}
	return
}
