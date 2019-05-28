package operations

import (
	"fmt"
	"sort"

	"github.com/dave/jennifer/jen"
)

// GeneratorProperties contains the values used by the generator to generate Index-related classes
type GeneratorProperties struct {
	Pkg            string
	Object         string
	Singular       string
	Plural         string
	IDFunc         string
	SearchCategory string
	WriteOptions   bool
	OptionsPath    string
	ObjectPathName string
	Tag            string
}

// methodGenerator generates an interface and implementation for a specific kind of DB operation.
type methodGenerator func(props GeneratorProperties) (interfaceMethod jen.Code, implementation jen.Code)

var (
	supportedMethods = make(map[string]methodGenerator)
)

func getOpNames() []string {
	// get deterministically sorted op names so the style checker won't complain about stale generated code
	opNames := make([]string, 0, len(supportedMethods))
	for opName := range supportedMethods {
		opNames = append(opNames, opName)
	}
	sort.Strings(opNames)
	return opNames
}

// GenerateInterfaceAndImplementation generates the interface definition and the implementation for the given DB operation.
func GenerateInterfaceAndImplementation(props GeneratorProperties) ([]jen.Code, []jen.Code) {
	interfaceMethods := make([]jen.Code, 0, len(supportedMethods))
	implementations := make([]jen.Code, 0, len(supportedMethods))
	for _, opName := range getOpNames() {
		method, ok := supportedMethods[opName]
		if !ok {
			panic(fmt.Sprintf("UNEXPECTED: method %s not found", opName))
		}
		interfaceMethod, implementation := method(props)
		interfaceMethods = append(interfaceMethods, interfaceMethod)
		implementations = append(implementations, implementation)
	}
	return interfaceMethods, implementations
}

// IsSupported returns whether the given opName is supported.
func IsSupported(opName string) bool {
	_, ok := supportedMethods[opName]
	return ok
}
