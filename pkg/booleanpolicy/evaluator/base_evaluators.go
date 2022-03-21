package evaluator

import (
	"container/heap"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/mapeval"
	"github.com/stackrox/rox/pkg/booleanpolicy/evaluator/pathutil"
	"github.com/stackrox/rox/pkg/booleanpolicy/query"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/readable"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stackrox/rox/pkg/search/predicate/basematchers"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	timestampPtrType = reflect.TypeOf((*types.Timestamp)(nil))
)

// A baseEvaluator is an evaluator that operates on an individual field at the leaf of an object.
type baseEvaluator interface {
	Evaluate(*pathutil.Path, reflect.Value) (*fieldResult, bool)
}

type baseEvaluatorFunc func(*pathutil.Path, reflect.Value) (*fieldResult, bool)

func (f baseEvaluatorFunc) Evaluate(path *pathutil.Path, value reflect.Value) (*fieldResult, bool) {
	return f(path, value)
}

func createBaseEvaluator(fieldName string, fieldType reflect.Type, values []string, negate bool, operator query.Operator, matchAll bool) (baseEvaluator, error) {
	lenValues := len(values)
	if (matchAll && lenValues > 0) || (!matchAll && lenValues == 0) {
		return nil, errors.New("invalid number of values")
	}
	if lenValues > 1 && operator != query.Or && operator != query.And {
		return nil, errors.Errorf("invalid operator: %s", operator)
	}
	generatorForKind, err := getMatcherGeneratorForKind(fieldType.Kind())
	if err != nil {
		return nil, err
	}

	if matchAll {
		m, err := generatorForKind("", fieldType, matchAll)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid match all generator for field %s", fieldName)
		}
		return baseEvaluatorFunc(func(path *pathutil.Path, instance reflect.Value) (*fieldResult, bool) {
			valuesAndMatches := m(instance)
			var values []string
			if len(valuesAndMatches) == 0 {
				return nil, false
			}
			for _, valueAndMatch := range valuesAndMatches {
				values = append(values, valueAndMatch.value)
			}
			return fieldResultWithSingleMatch(fieldName, path, values...), true
		}), nil
	}

	baseMatchers := make([]baseMatcherAndExtractor, 0, len(values))
	for _, value := range values {
		m, err := generatorForKind(value, fieldType, false)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid value: %s for field %s", value, fieldName)
		}
		baseMatchers = append(baseMatchers, m)
	}

	if negate {
		return combineMatchersIntoEvaluatorNegated(fieldName, baseMatchers, operator), nil
	}
	return combineMatchersIntoEvaluator(fieldName, baseMatchers, operator), nil
}

func combineMatchersIntoEvaluator(fieldName string, matchers []baseMatcherAndExtractor, operator query.Operator) baseEvaluator {
	return baseEvaluatorFunc(func(path *pathutil.Path, instance reflect.Value) (*fieldResult, bool) {
		matchingValues := set.NewStringSet()
		var matches []string
		for _, m := range matchers {
			valuesAndMatches := m(instance)
			// This means there were no values.
			if len(valuesAndMatches) == 0 {
				return nil, false
			}
			var atLeastOneSuccess bool
			for _, valueAndMatch := range valuesAndMatches {
				if valueAndMatch.matched {
					if matchingValues.Add(valueAndMatch.value) {
						matches = append(matches, valueAndMatch.value)
					}
					atLeastOneSuccess = true
				}
			}
			// If not matched, and it's an And, then we can early exit.
			if !atLeastOneSuccess && operator == query.And {
				return nil, false
			}
		}
		if matchingValues.Cardinality() == 0 {
			return nil, false
		}
		return fieldResultWithSingleMatch(fieldName, path, matches...), true
	})
}

func combineMatchersIntoEvaluatorNegated(fieldName string, matchers []baseMatcherAndExtractor, operator query.Operator) baseEvaluator {
	return baseEvaluatorFunc(func(path *pathutil.Path, instance reflect.Value) (*fieldResult, bool) {
		matchingValues := set.NewStringSet()
		var matches []string
		var atLeastOneMatcherDidNotMatch bool
		for _, m := range matchers {
			valuesAndMatches := m(instance)
			// This means there were no values.
			if len(valuesAndMatches) == 0 {
				return nil, false
			}
			var atLeastOneMatch bool
			for _, valueAndMatch := range valuesAndMatches {
				if valueAndMatch.matched {
					atLeastOneMatch = true
				} else {
					if val := valueAndMatch.value; matchingValues.Add(val) {
						matches = append(matches, val)
					}
				}
			}

			// If it matched, and it's an Or, then we can early exit.
			// Since we're negating, this check is correct by de Morgan's law.
			// !(A OR B) <=> !A AND !B, therefore if operator is OR and A _does_ match,
			// we can conclude that !A is false => !A AND !B is false => !(A OR B) is false.
			if atLeastOneMatch && operator == query.Or {
				return nil, false
			}
			if !atLeastOneMatch {
				atLeastOneMatcherDidNotMatch = true
			}
		}
		if !atLeastOneMatcherDidNotMatch {
			return nil, false
		}
		return fieldResultWithSingleMatch(fieldName, path, matches...), true
	})
}

func getMatcherGeneratorForKind(kind reflect.Kind) (baseMatcherGenerator, error) {
	switch kind {
	case reflect.String:
		return generateStringMatcher, nil
	case reflect.Ptr:
		return generatePtrMatcher, nil
	case reflect.Array, reflect.Slice:
		return generateSliceMatcher, nil
	case reflect.Map:
		return generateMapMatcher, nil
	case reflect.Bool:
		return generateBoolMatcher, nil
	case reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8, reflect.Int:
		return generateIntMatcher, nil
	case reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8, reflect.Uint:
		return generateUintMatcher, nil
	case reflect.Float64, reflect.Float32:
		return generateFloatMatcher, nil
	default:
		return nil, errors.Errorf("invalid kind for base query: %s", kind)
	}
}

type baseMatcherGenerator func(string, reflect.Type, bool) (baseMatcherAndExtractor, error)

// A baseMatcherAndExtractor takes a value of a given type, extracts a human-readable string value
// and returns whether it matched or not.
// IMPORTANT: in every valueMatchedPair, value _must_ be returned even if _matched_ is false,
// since that enables us to get the value out even if the caller is going to negate this query.
type baseMatcherAndExtractor func(reflect.Value) []valueMatchedPair

type valueMatchedPair struct {
	value   string
	matched bool
}

func fieldResultWithSingleMatch(fieldName string, path *pathutil.Path, values ...string) *fieldResult {
	return &fieldResult{map[string][]Match{fieldName: {{Path: path, Values: values}}}}
}

func generateStringMatcher(value string, _ reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(string) bool
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		var err error
		baseMatcher, err = basematchers.ForString(value)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		if instance.Kind() != reflect.String {
			return nil
		}
		asStr := instance.String()
		return []valueMatchedPair{{value: asStr, matched: matchAll || baseMatcher(asStr)}}
	}, nil
}

func generateSliceMatcher(value string, fieldType reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	underlyingType := fieldType.Elem()
	matcherGenerator, err := getMatcherGeneratorForKind(underlyingType.Kind())
	if err != nil {
		return nil, err
	}
	subMatcher, err := matcherGenerator(value, underlyingType, matchAll)
	if err != nil {
		return nil, err
	}
	return func(instance reflect.Value) []valueMatchedPair {
		length := instance.Len()
		if length == 0 {
			// An empty slice matches no queries, but we want to bubble this up,
			// for callers that are negating.
			return []valueMatchedPair{{value: "<empty>", matched: matchAll}}
		}
		valuesAndMatches := make([]valueMatchedPair, 0, length)
		for i := 0; i < length; i++ {
			valuesAndMatches = append(valuesAndMatches, subMatcher(instance.Index(i))...)
		}
		return valuesAndMatches
	}, nil
}

func generateTimestampMatcher(value string, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(*types.Timestamp) bool
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		if value != search.NullString {
			var err error
			baseMatcher, err = basematchers.ForTimestamp(value)
			if err != nil {
				return nil, err
			}
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		ts, ok := instance.Interface().(*types.Timestamp)
		if !ok {
			return nil
		}
		if ts == nil {
			if matchAll || value == search.NullString {
				return []valueMatchedPair{{value: "<empty timestamp>", matched: true}}
			}
			return nil
		}
		return []valueMatchedPair{{value: readable.ProtoTime(ts), matched: matchAll || (value != "-" && baseMatcher(ts))}}
	}, nil
}

func generatePtrMatcher(value string, fieldType reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	// Special case for pointer to timestamp.
	if fieldType == timestampPtrType {
		return generateTimestampMatcher(value, matchAll)
	}
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	underlyingType := fieldType.Elem()
	var subMatcher func(reflect.Value) []valueMatchedPair
	matcherGenerator, err := getMatcherGeneratorForKind(underlyingType.Kind())
	if err != nil {
		// If testing for nil, the submatcher is not required, so this is okay.
		if !matchAll && value != search.NullString {
			return nil, err
		}
		matcherGenerator = nil
	}
	if matcherGenerator != nil {
		var err error
		subMatcher, err = matcherGenerator(value, underlyingType, matchAll)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		if instance.IsNil() {
			return []valueMatchedPair{{value: "<nil>", matched: matchAll || value == search.NullString}}
		}
		var subMatches []valueMatchedPair
		if subMatcher != nil {
			subMatches = subMatcher(instance.Elem())
		} else {
			subMatches = []valueMatchedPair{{value: "<non-nil>"}}
		}
		// If the value is null, and the pointer is not nil, it did not match.
		// So just use the values from the subMatcher but always set matched
		// to false.
		if !matchAll && value == search.NullString {
			for i := range subMatches {
				subMatches[i].matched = false
			}
		}
		return subMatches
	}, nil
}

func generateBoolMatcher(value string, _ reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(bool) bool
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		var err error
		baseMatcher, err = basematchers.ForBool(value)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		if instance.Kind() != reflect.Bool {
			return nil
		}
		asBool := instance.Bool()
		return []valueMatchedPair{{value: fmt.Sprintf("%t", asBool), matched: matchAll || baseMatcher(asBool)}}
	}, nil
}

func generateIntMatcher(value string, fieldType reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	if enum, ok := reflect.Zero(fieldType).Interface().(protoreflect.ProtoEnum); ok {
		return generateEnumMatcher(value, enum, matchAll)
	}
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	var baseMatcher func(int64) bool
	if !matchAll {
		var err error
		baseMatcher, err = basematchers.ForInt(value)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		switch instance.Kind() {
		case reflect.Int, reflect.Int64, reflect.Int32, reflect.Int16, reflect.Int8:
			asInt := instance.Int()
			return []valueMatchedPair{{value: fmt.Sprintf("%d", asInt), matched: matchAll || baseMatcher(asInt)}}
		}
		return nil
	}, nil
}

func generateUintMatcher(value string, _ reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(uint64) bool
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		var err error
		baseMatcher, err = basematchers.ForUint(value)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		switch instance.Kind() {
		case reflect.Uint, reflect.Uint64, reflect.Uint32, reflect.Uint16, reflect.Uint8:
			asUint := instance.Uint()
			return []valueMatchedPair{{value: fmt.Sprintf("%d", asUint), matched: matchAll || baseMatcher(asUint)}}
		}
		return nil
	}, nil
}

func generateFloatMatcher(value string, _ reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(float64) bool
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		var err error
		baseMatcher, err = basematchers.ForFloat(value)
		if err != nil {
			return nil, err
		}
	}
	return func(instance reflect.Value) []valueMatchedPair {
		switch instance.Kind() {
		case reflect.Float32, reflect.Float64:
			asFloat := instance.Float()
			return []valueMatchedPair{{value: readable.Float(asFloat, 3), matched: matchAll || baseMatcher(asFloat)}}
		}
		return nil
	}, nil
}

func isUnsetEnum(value string) bool {
	return strings.HasPrefix(strings.ToLower(value), "unset")
}

func generateEnumMatcher(value string, enumRef protoreflect.ProtoEnum, matchAll bool) (baseMatcherAndExtractor, error) {
	var baseMatcher func(int64) bool
	var numberToName map[int32]string
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}
	if !matchAll {
		var err error
		baseMatcher, numberToName, err = basematchers.ForEnum(value, enumRef)
		if err != nil {
			return nil, err
		}
	} else {
		enumDesc, err := protoreflect.GetEnumDescriptor(enumRef)
		if err != nil {
			return nil, err
		}
		_, numberToName = basematchers.MapEnumValues(enumDesc)
	}
	return func(instance reflect.Value) []valueMatchedPair {
		if instance.Kind() != reflect.Int32 {
			return nil
		}
		asInt := instance.Int()
		matchedValue := numberToName[int32(asInt)]
		if matchedValue == "" {
			utils.Should(errors.Errorf("enum query matched (%v), but no value in numberToName (%v) (got number: %d)",
				value, numberToName, asInt))
			matchedValue = strconv.Itoa(int(asInt))
		}
		// Treat an unset enum as an undefined value -- it matches no numeric query.
		if !matchAll && asInt == 0 && isUnsetEnum(matchedValue) {
			return nil
		}
		return []valueMatchedPair{{value: matchedValue, matched: matchAll || baseMatcher(asInt)}}
	}, nil
}

const (
	maxNumberOfKeyValuePairs = 3
	maxValueLength           = 64
)

func generateMapMatcher(value string, typ reflect.Type, matchAll bool) (baseMatcherAndExtractor, error) {
	if matchAll && value != "" {
		return nil, errors.New("non-empty value for matchAll")
	}

	baseMatcher, err := mapeval.Matcher(value, typ)
	if err != nil {
		return nil, err
	}

	return func(instance reflect.Value) []valueMatchedPair {
		if instance.Kind() != reflect.Map {
			return nil
		}

		iter := instance.MapRange()
		mapResult, matched := baseMatcher(iter, maxNumberOfKeyValuePairs)
		var res string
		if matchAll {
			res = printKVs(mapResult.KeyValues, mapResult.NumElements)
		} else {
			res = printMatchingKVsOrKVs(mapResult, mapResult.NumElements)
		}

		return []valueMatchedPair{{value: res, matched: matchAll || matched}}
	}, nil
}

func printKVs(kvPairs heap.Interface, totalElements int) string {
	var asSlice []*mapeval.KeyValue
	if length := kvPairs.Len(); length > 0 {
		asSlice = make([]*mapeval.KeyValue, length)
		for i := 0; i < length; i++ {
			// The heap is a max-heap, but we want to sort it in increasing value, so we fill
			// in backwards.
			asSlice[length-1-i] = heap.Pop(kvPairs).(*mapeval.KeyValue)
		}
	}
	return printKVsFromSortedSlice(asSlice, totalElements)
}

func printKVsFromSortedSlice(kvPairs []*mapeval.KeyValue, totalElements int) string {
	if len(kvPairs) == 0 {
		return "<empty>"
	}

	var sb strings.Builder
	for i, kvPair := range kvPairs {
		sb.WriteString(fmt.Sprintf("%s=%s", kvPair.Key, stringutils.Truncate(kvPair.Value, maxValueLength)))
		if i < len(kvPairs)-1 {
			sb.WriteString(", ")
		}
	}
	if numExtraPairs := totalElements - len(kvPairs); numExtraPairs > 0 {
		sb.WriteString(fmt.Sprintf(" and %d more", numExtraPairs))
	}
	return sb.String()
}

func printMatchingKVsOrKVs(value *mapeval.MatcherResults, totalElements int) string {
	if len(value.MatchingKeyValues) > 0 {
		asSlice := make([]*mapeval.KeyValue, 0, len(value.MatchingKeyValues))
		for k := range value.MatchingKeyValues {
			asSlice = append(asSlice, k)
		}
		sort.Slice(asSlice, func(i, j int) bool {
			return asSlice[i].Key < asSlice[j].Key
		})
		return printKVsFromSortedSlice(asSlice, len(asSlice))
	}

	return printKVs(value.KeyValues, totalElements)
}
