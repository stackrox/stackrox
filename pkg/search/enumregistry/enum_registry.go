package enumregistry

import (
	"regexp"
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var (
	// enumRegistry holds path -> enum maps that we have constructed from the protos
	enumMap        map[string]map[string]int32
	reverseEnumMap map[string]map[int32]string
)

// Add takes in a path and an enum descriptor and creates a path -> map[string enum]int32 value.
func Add(path string, enumDescriptor *descriptor.EnumDescriptorProto) {
	if _, ok := enumMap[path]; !ok {
		enumMap[path] = make(map[string]int32)
		reverseEnumMap[path] = make(map[int32]string)
	}
	subMap := enumMap[path]
	subReverseMap := reverseEnumMap[path]
	for _, v := range enumDescriptor.GetValue() {
		subMap[strings.ToLower(*v.Name)] = *v.Number
		subReverseMap[*v.Number] = *v.Name
	}
}

// GetComplement takes in a field path and a string to evaluate against, and returns the int32 form.
// of the complement of matching enums.
func GetComplement(fieldPath string, s string) []int32 {
	lowerS := strings.ToLower(s)
	return get(fieldPath, func(k string) bool {
		return !strings.HasPrefix(k, lowerS)
	})
}

// Get takes in a field path and a string to evaluate against and returns the int32 form of any matching enums.
func Get(fieldPath string, s string) []int32 {
	lowerS := strings.ToLower(s)
	return get(fieldPath, func(k string) bool {
		return strings.HasPrefix(k, lowerS)
	})
}

// GetExactMatches takes in a field path and a string and returns the int32 forms of any exact matches.
func GetExactMatches(fieldPath, s string) []int32 {
	lowerS := strings.ToLower(s)
	return get(fieldPath, func(k string) bool {
		return lowerS == k
	})
}

// GetComplementByExactMatches takes in a field path and a string and returns the int32 forms
// of all values that are not an exact match.
func GetComplementByExactMatches(fieldPath, s string) []int32 {
	lowerS := strings.ToLower(s)
	return get(fieldPath, func(k string) bool {
		return lowerS != k
	})
}

// GetValuesMatchingRegex takes in a field path, and a regex, and returns the int32 form of any matching enums.
func GetValuesMatchingRegex(fieldPath string, re *regexp.Regexp) []int32 {
	return get(fieldPath, re.MatchString)
}

// GetComplementOfValuesMatchingRegex takes in a field path, and a regex, and returns the int32 form of any enums
// that don't match.
func GetComplementOfValuesMatchingRegex(fieldPath string, re *regexp.Regexp) []int32 {
	return get(fieldPath, func(k string) bool {
		return !re.MatchString(k)
	})
}

func get(fieldPath string, include func(string) bool) []int32 {
	m, ok := enumMap[fieldPath]
	if !ok {
		return nil
	}
	var matches []int32
	for k, v := range m {
		if include(k) {
			matches = append(matches, v)
		}
	}
	return matches
}

// Lookup takes in a field path and an int32 and returns the string version of the proto value
func Lookup(fieldPath string, val int32) string {
	m, ok := reverseEnumMap[fieldPath]
	if !ok {
		return ""
	}
	return m[val]
}

// IsEnum takes in a fieldpath and returns whether or not it's an enum
func IsEnum(fieldPath string) bool {
	_, ok := enumMap[fieldPath]
	return ok
}

func init() {
	enumMap = make(map[string]map[string]int32)
	reverseEnumMap = make(map[string]map[int32]string)

}
