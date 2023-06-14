package basematchers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/parse"
	"github.com/stackrox/rox/pkg/protoreflect"
	"github.com/stackrox/rox/pkg/regexutils"
	"github.com/stackrox/rox/pkg/search"
)

// ForString returns a matcher for a string.
func ForString(value string) (func(string) bool, error) {
	negated := strings.HasPrefix(value, search.NegationPrefix)
	if negated {
		value = strings.TrimPrefix(value, search.NegationPrefix)
	}
	if strings.HasPrefix(value, search.RegexPrefix) {
		value = strings.TrimPrefix(value, search.RegexPrefix)
		return forStringRegexMatch(value, negated)
	} else if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) && len(value) > 1 {
		return forStringExactMatch(value[1:len(value)-1], negated)
	}
	return forStringPrefixMatch(value, negated)
}

// ForFloat returns a matcher for a float64.
func ForFloat(value string) (func(float64) bool, error) {
	cmpStr, value := parseNumericPrefix(value)
	comparator, err := floatComparator(cmpStr)
	if err != nil {
		return nil, err
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return nil, err
	}
	return func(instance float64) bool {
		return comparator(instance, floatValue)
	}, nil
}

// ForUint returns a matcher for a uint.
func ForUint(value string) (func(uint64) bool, error) {
	cmpStr, value := parseNumericPrefix(value)
	comparator, err := uintComparator(cmpStr)
	if err != nil {
		return nil, err
	}

	uIntValue, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return func(instance uint64) bool {
		return comparator(instance, uIntValue)
	}, nil
}

// ForInt returns a matcher for a uint.
func ForInt(value string) (func(int64) bool, error) {
	cmpStr, value := parseNumericPrefix(value)
	comparator, err := intComparator(cmpStr)
	if err != nil {
		return nil, err
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return nil, err
	}
	return func(instance int64) bool {
		return comparator(instance, intValue)
	}, nil
}

func stringMatchEnums(value string, enumMap map[int32]string) (func(int64) bool, error) {
	matcher, err := ForString(value)
	if err != nil {
		return nil, err
	}
	return func(v int64) bool {
		return matcher(enumMap[int32(v)])
	}, nil
}

// ForEnum returns a matcher for an enum.
// The matcher takes a query against the string version of the enum,
// and matches it against the int value which is what is actually stored.
func ForEnum(value string, enumRef protoreflect.ProtoEnum) (func(int64) bool, map[int32]string, error) {
	// Map the enum strings to integer values.
	enumDesc, err := protoreflect.GetEnumDescriptor(enumRef)
	if err != nil {
		return nil, nil, err
	}
	nameToNumber, numberToName := MapEnumValues(enumDesc)

	// "" and search.WildcardString imply matching all
	if value == "" || value == search.WildcardString {
		return func(int64) bool {
			return true
		}, numberToName, nil
	}

	// Get the comparator if needed.
	cmpStr, value := parseNumericPrefix(value)

	// Translate input value to an int if needed.
	var int64Value int64
	int32Value, hasIntValue := nameToNumber[strings.ToLower(value)]
	if hasIntValue {
		int64Value = int64(int32Value)
	} else {
		matcher, err := stringMatchEnums(value, numberToName)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unrecognized enum value: %s in %+v", value, nameToNumber)
		}
		return matcher, numberToName, nil
	}

	// Generate the comparator for the integer values.
	comparator, err := intComparator(cmpStr)
	if err != nil {
		return nil, nil, err
	}
	return func(instance int64) bool {
		return comparator(instance, int64Value)
	}, numberToName, nil
}

// ForBool returns a matcher for a bool value.
func ForBool(value string) (func(bool) bool, error) {
	boolValue, err := parse.FriendlyParseBool(value)
	if err != nil {
		return nil, err
	}
	return func(instance bool) bool {
		return instance == boolValue
	}, nil
}

// ForTimestamp returns a matcher for a proto timestamp type.
func ForTimestamp(value string) (func(*types.Timestamp) bool, error) {
	if value == "-" {
		return func(instance *types.Timestamp) bool {
			return instance == nil
		}, nil
	}
	cmpStr, value := parseNumericPrefix(value)

	timestampValue, durationValue, err := parseTimestamp(value)
	if err != nil {
		return nil, err
	}
	comparator, err := timestampComparator(cmpStr)
	if err != nil {
		return nil, err
	}
	actualComparator := comparator
	// If we're using a duration value, we need to invert the query.
	// This is because, for example, >90d means more than 90 days ago,
	// which means <=(ts of time.Now().Add(-90days).
	if durationValue != nil {
		actualComparator = func(instance, value *types.Timestamp) bool {
			return !comparator(instance, value)
		}
	}

	return func(instance *types.Timestamp) bool {
		// This has to be done inside the closure, since we want to take time.Now() at evaluation time,
		// not at build time.
		var ts *types.Timestamp
		if timestampValue != nil {
			ts = timestampValue
		} else if durationValue != nil {
			ts, err = types.TimestampProto(time.Now().Add(-*durationValue))
			if err != nil {
				return false
			}
		}

		// Value is NOT "-" here, that case is handled above.
		if instance == nil {
			return false
		}
		return actualComparator(instance, ts)
	}, nil
}

// MapEnumValues provides mappings between enum string name and enum number
func MapEnumValues(enumDesc *descriptor.EnumDescriptorProto) (nameToNumber map[string]int32, numberToName map[int32]string) {
	nameToNumber = make(map[string]int32, len(enumDesc.GetValue()))
	numberToName = make(map[int32]string, len(enumDesc.GetValue()))
	for _, v := range enumDesc.GetValue() {
		lName := strings.ToLower(v.GetName())
		nameToNumber[lName] = v.GetNumber()
		numberToName[v.GetNumber()] = lName
	}
	return
}

func forStringRegexMatch(regex string, negated bool) (func(string) bool, error) {
	matcher, err := regexutils.CompileWholeStringMatcher(regex, regexutils.Flags{CaseInsensitive: true})
	if err != nil {
		return nil, errors.Wrapf(err, "invalid regex: %q", regex)
	}

	return func(instance string) bool {
		// matched != negated is equivalent to (matched XOR negated), which is what we want here
		return matcher.MatchWholeString(instance) != negated
	}, nil
}

func forStringExactMatch(value string, negated bool) (func(string) bool, error) {
	lowerValue := strings.ToLower(value)
	return func(instance string) bool {
		// matched != negated is equivalent to (matched XOR negated), which is what we want here
		return (lowerValue == strings.ToLower(instance)) != negated
	}, nil
}

func forStringPrefixMatch(value string, negated bool) (func(string) bool, error) {
	lowerValue := strings.ToLower(value)
	return func(instance string) bool {
		// matched != negated is equivalent to (matched XOR negated), which is what we want here
		return (value == search.WildcardString || strings.HasPrefix(strings.ToLower(instance), lowerValue)) != negated
	}, nil
}
