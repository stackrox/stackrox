package enumregistry

import (
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var (
	// enumRegistry holds path -> enum maps that we have constructed from the protos
	enumMap        map[string]map[string]int32
	reverseEnumMap map[string]map[int32]string
)

// Add takes in a path and an enum descriptor and creates a path -> map[string enum]int32 value
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

// Get takes in a field path and a string to evaluate against and returns the int32 form of any matching enums
func Get(fieldPath string, s string) []int32 {
	s = strings.ToLower(s)
	m, ok := enumMap[fieldPath]
	if !ok {
		return nil
	}
	var matches []int32
	for k, v := range m {
		if strings.HasPrefix(k, s) {
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
