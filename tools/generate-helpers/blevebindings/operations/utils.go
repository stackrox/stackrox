package operations

import (
	"path"
	"strings"

	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/rox/tools/generate-helpers/common/packagenames"
)

func renderFuncBStarIndexer() *Statement {
	return Func().Params(Id("b").Op("*").Id("indexerImpl"))
}

// MakeWrapperType takes a struct name and formats it like the index wrapper struct name
func MakeWrapperType(str string) string {
	if len(str) <= 1 {
		return strings.ToLower(str) + "Wrapper"
	}
	return strings.ToLower(str[:1]) + str[1:] + "Wrapper"
}

func metricLine(op, name string) *Statement {
	return Defer().Qual(packagenames.Metrics, "SetIndexOperationDurationTime").Call(Qual("time", "Now").Call(), Qual(packagenames.Ops, op), Lit(name))
}

func bIndex() *Statement {
	return Id("b").Dot("index")
}

func ifErrReturnError(statement *Statement) *Statement {
	return If(Err().Op(":=").Add(statement), Err().Op("!=").Nil()).Block(
		Return(Err()),
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
