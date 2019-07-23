package operations

import (
	"path"
	"strings"

	"github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/blevebindings/packagenames"
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

func metricLine(op, name string) *jen.Statement {
	return jen.Defer().Qual(packagenames.Metrics, "SetIndexOperationDurationTime").Call(jen.Qual("time", "Now").Call(), jen.Qual(packagenames.Ops, op), jen.Lit(name))
}

func bIndex() *jen.Statement {
	return jen.Id("b").Dot("index")
}

func ifErrReturnError(statement *jen.Statement) *jen.Statement {
	return jen.If(jen.Err().Op(":=").Add(statement), jen.Err().Op("!=").Nil()).Block(
		jen.Return(jen.Err()),
	)
}

// GenerateMappingGoPackage generates the go package corresponding to the mapping directory.
func GenerateMappingGoPackage(props GeneratorProperties) string {
	objectName := props.Object
	if props.ObjectPathName != "" {
		objectName = props.ObjectPathName
	}
	return path.Join(packagenames.RoxCentral, strings.ToLower(objectName), props.OptionsPath)
}
