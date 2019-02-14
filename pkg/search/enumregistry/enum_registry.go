package enumregistry

import (
	"strings"
	"sync"

	"github.com/golang/protobuf/protoc-gen-go/descriptor"
)

var (
	once sync.Once

	reg Registry
)

// Singleton returns the enum registry
func Singleton() Registry {
	once.Do(func() {
		reg = &enumRegistry{
			enumMap:        make(map[string]map[string]int32),
			reverseEnumMap: make(map[string]map[int32]string),
		}
	})
	return reg
}

// Registry is a registry of field paths to their relative int32 and string representations
type Registry interface {
	Add(path string, enumDescriptor *descriptor.EnumDescriptorProto)
	Get(fieldPath string, s string) []int32
	Lookup(fieldPath string, val int32) string
	IsEnum(fieldPath string) bool
}

//go:generate mockgen-wrapper Registry

// enumRegistry holds path -> enum maps that we have constructed from the protos
type enumRegistry struct {
	enumMap        map[string]map[string]int32
	reverseEnumMap map[string]map[int32]string
}

// Add takes in a path and an enum descriptor and creates a path -> map[string enum]int32 value
func (e *enumRegistry) Add(path string, enumDescriptor *descriptor.EnumDescriptorProto) {
	if _, ok := e.enumMap[path]; !ok {
		e.enumMap[path] = make(map[string]int32)
		e.reverseEnumMap[path] = make(map[int32]string)
	}
	subMap := e.enumMap[path]
	subReverseMap := e.reverseEnumMap[path]
	for _, v := range enumDescriptor.GetValue() {
		subMap[strings.ToLower(*v.Name)] = *v.Number
		subReverseMap[*v.Number] = *v.Name
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

// Lookup takes in a field path and an int32 and returns the string version of the proto value
func (e *enumRegistry) Lookup(fieldPath string, val int32) string {
	m, ok := e.reverseEnumMap[fieldPath]
	if !ok {
		return ""
	}
	return m[val]
}

// IsEnum takes in a fieldpath and returns whether or not it's an enum
func (e *enumRegistry) IsEnum(fieldPath string) bool {
	_, ok := e.enumMap[fieldPath]
	return ok
}
