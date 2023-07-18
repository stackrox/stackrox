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

// GenerateMappingGoSubPackageWithinCentral generates the go package corresponding to the mapping directory,
// but stripping out the prefix for central.
func GenerateMappingGoSubPackageWithinCentral(props GeneratorProperties) string {
	objectName := props.Object
	if props.ObjectPathName != "" {
		objectName = props.ObjectPathName
	}
	return path.Join(strings.ToLower(objectName), props.OptionsPath)
}

// GenerateMappingGoPackage generates the go package corresponding to the mapping directory.
func GenerateMappingGoPackage(props GeneratorProperties) string {
	return path.Join(packagenames.RoxCentral, GenerateMappingGoSubPackageWithinCentral(props))
}
