package operations

import (
	"strings"

	"github.com/dave/jennifer/jen"
)

func renderFuncBStarIndexer() *jen.Statement {
	return jen.Func().Params(jen.Id("b").Op("*").Id("indexerImpl"))
}

// MakeWrapperType takes a struct name and formats it like the index wrapper struct name
func MakeWrapperType(str string) string {
	if len(str) <= 1 {
		return strings.ToLower(str) + "Wrapper"
	}
	return strings.ToLower(str[:1]) + str[1:] + "Wrapper"
}
