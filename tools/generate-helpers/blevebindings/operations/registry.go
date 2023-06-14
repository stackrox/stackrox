package operations

import (
	"fmt"
	"sort"

	. "github.com/dave/jennifer/jen"
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
	NeedsTxManager bool

	GenerateMockIndexer          bool
	MockgenWrapperExecutablePath string
}

// methodGenerator generates an interface and implementation for a specific kind of DB operation.
type methodGenerator func(props GeneratorProperties) (interfaceMethod Code, implementation Code)

var (
	supportedMethods    = make(map[string]methodGenerator)
	supportedTxnMethods = make(map[string]methodGenerator)
)

func getOpNames(methods map[string]methodGenerator) []string {
	// get deterministically sorted op names so the style checker won't complain about stale generated code
	opNames := make([]string, 0, len(methods))
	for opName := range methods {
		opNames = append(opNames, opName)
	}
	sort.Strings(opNames)
	return opNames
}

// GenerateInterfaceAndImplementation generates the interface definition and the implementation for the given DB operation.
func GenerateInterfaceAndImplementation(props GeneratorProperties) []Code {
	interfaceMethods := make([]Code, 0, len(supportedMethods))
	for _, opName := range getOpNames(supportedMethods) {
		method, ok := supportedMethods[opName]
		if !ok {
			panic(fmt.Sprintf("UNEXPECTED: method %s not found", opName))
		}
		interfaceMethod, _ := method(props)
		interfaceMethods = append(interfaceMethods, interfaceMethod)
	}
	if props.NeedsTxManager {
		for _, opName := range getOpNames(supportedTxnMethods) {
			method, ok := supportedTxnMethods[opName]
			if !ok {
				panic(fmt.Sprintf("UNEXPECTED: method %s not found", opName))
			}
			interfaceMethod, _ := method(props)
			interfaceMethods = append(interfaceMethods, interfaceMethod)
		}
	}

	return interfaceMethods
}
