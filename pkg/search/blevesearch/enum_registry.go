package blevesearch

import (
	"strings"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// enumRegistry holds path -> enum maps that we have constructed from the protos
type enumRegistry struct {
	enumMap map[string]map[string]int32
}

func newEnumRegistry() *enumRegistry {
	return &enumRegistry{
		enumMap: make(map[string]map[string]int32),
	}
}

// Add takes in a path and an enum descriptor and creates a path -> map[string enum]int32 value
func (e *enumRegistry) Add(path string, enumDescriptor *descriptor.EnumDescriptorProto) {
	if _, ok := e.enumMap[path]; !ok {
		e.enumMap[path] = make(map[string]int32)
	}
	subMap := e.enumMap[path]
	for _, v := range enumDescriptor.GetValue() {
		subMap[strings.ToLower(*v.Name)] = *v.Number
	}
}

// Get takes in a field path and a string to evaluate against and returns the int32 form of any matching enums
func (e *enumRegistry) Get(fieldPath string, s string) []int32 {
	s = strings.ToLower(s)
	m, ok := e.enumMap[fieldPath]
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
