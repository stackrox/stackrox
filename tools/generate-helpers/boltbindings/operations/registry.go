package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

// GeneratorProperties contains the values used by the generator to generate Store-related classes
type GeneratorProperties struct {
	Pkg          string
	Object       string
	Singular     string
	Plural       string
	IDFunc       string
	IDField      string
	BucketName   string
	GetExists    bool
	DeleteExists bool
}

// methodGenerator generates an interface and implementation for a specific kind of DB operation.
type methodGenerator func(props *GeneratorProperties) (interfaceMethod Code, implementation Code)

var (
	supportedMethods = make(map[string]methodGenerator)
)

// RenderSupportedMethods renders a comma-separated string with the names of the supported methods.
func RenderSupportedMethods() string {
	methods := make([]string, 0, len(supportedMethods))
	for method := range supportedMethods {
		methods = append(methods, method)
	}
	return strings.Join(methods, ", ")
}

// GenerateInterfaceAndImplementation generates the interface definition and the implementation for the given DB operation.
func GenerateInterfaceAndImplementation(opName string, props *GeneratorProperties) (interfaceMethod Code, implementation Code) {
	method, ok := supportedMethods[opName]
	if !ok {
		panic(fmt.Sprintf("UNEXPECTED: method %s not found", opName))
	}
	return method(props)
}

// IsSupported returns whether the given opName is supported.
func IsSupported(opName string) bool {
	_, ok := supportedMethods[opName]
	return ok
}
