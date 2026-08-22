package enumregistry

import (
	"regexp"
	"strings"

	"google.golang.org/protobuf/types/descriptorpb"
)

var (
	// enumRegistry holds path -> enum maps that we have constructed from the protos
	enumMap        map[string]map[string]int32
	reverseEnumMap map[string]map[int32]string
)

// Add takes in a path and an enum descriptor and creates a path -> map[string enum]int32 value.
func Add(path string, enumDescriptor *descriptorpb.EnumDescriptorProto) {
	if _, ok := enumMap[path]; !ok {
		enumMap[path] = make(map[string]int32)
		reverseEnumMap[path] = make(map[int32]string)
	}
	subMap := enumMap[path]
	subReverseMap := reverseEnumMap[path]
	for _, v := range enumDescriptor.GetValue() {
		subMap[strings.ToLower(v.GetName())] = v.GetNumber()
		subReverseMap[v.GetNumber()] = v.GetName()
	}
}

// AddValues registers enum values for a field path using a pre-computed name→number map.
// Names in the values map should be in their ORIGINAL case (e.g., "UNSET", "LOW").
// The forward lookup map stores lowercased keys; the reverse map stores original-case values.
func AddValues(path string, values map[string]int32) {
	if _, ok := enumMap[path]; !ok {
		enumMap[path] = make(map[string]int32)
		reverseEnumMap[path] = make(map[int32]string)
	}
	for name, num := range values {
		enumMap[path][strings.ToLower(name)] = num
		reverseEnumMap[path][num] = name
	}
}

// Snapshot returns a deep copy of the current enum registry state.
// The returned map is path → (original_case_name → number), suitable
// for round-tripping through AddValues.
func Snapshot() map[string]map[string]int32 {
	result := make(map[string]map[string]int32, len(reverseEnumMap))
	for path, values := range reverseEnumMap {
		pathCopy := make(map[string]int32, len(values))
		for num, name := range values {
			pathCopy[name] = num
		}
		result[path] = pathCopy
	}
	return result
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
