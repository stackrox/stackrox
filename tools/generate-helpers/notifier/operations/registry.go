package operations

import (
	"fmt"
	"strings"

	. "github.com/dave/jennifer/jen"
)

// GeneratorProperties contains the values used by the generator to generate Store-related classes
type GeneratorProperties struct {
	Pkg    string
	Object string
}

// methodGenerator generates an interface and implementation for a specific kind of DB operation.
type methodGenerator interface {
	generate(props *GeneratorProperties) (signatures []Code, variables []Code, implementations []Code)
}

var (
	supportedMethods = map[string]methodGenerator{
		"add": &addAndNotifyGenerator{
			First: "Add",
			Third: "Adds",
			Past:  "Added",
		},
		"update": &addAndNotifyGenerator{
			First: "Update",
			Third: "Updates",
			Past:  "Updated",
		},
		"upsert": &addAndNotifyGenerator{
			First: "Upsert",
			Third: "Upserts",
			Past:  "Upserted",
		},
		"delete": &addAndNotifyGenerator{
			First: "Delete",
			Third: "Deletes",
			Past:  "Deleted",
		},
	}
)

// RenderSupportedMethods renders a comma-separated string with the names of the supported methods.
func RenderSupportedMethods() string {
	methods := make([]string, 0, len(supportedMethods))
	for method := range supportedMethods {
		methods = append(methods, method)
	}
	return strings.Join(methods, ", ")
}

// GenerateSignaturesAndImplementations generates the interface/signatures of the functions, the member variables of
// the implementating struct, and the implementation of each function.
func GenerateSignaturesAndImplementations(opName string, props *GeneratorProperties) (signatures []Code, variables []Code, implementations []Code) {
	generator, ok := supportedMethods[opName]
	if !ok {
		panic(fmt.Sprintf("UNEXPECTED: method %s not found", opName))
	}
	return generator.generate(props)
}

// IsSupported returns whether the given opName is supported.
func IsSupported(opName string) bool {
	_, ok := supportedMethods[opName]
	return ok
}
