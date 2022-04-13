package common

import (
	. "github.com/dave/jennifer/jen"
	"github.com/stackrox/stackrox/tools/generate-helpers/common/packagenames"
)

// RenderFuncSStarStore renders func (s *store)
func RenderFuncSStarStore() *Statement {
	return Func().Params(Id("s").Op("*").Id("store"))
}

// RenderBoltMetricLine generates a metric line for bolt operations.
func RenderBoltMetricLine(op, name string) *Statement {
	return Defer().Qual(packagenames.Metrics, "SetBoltOperationDurationTime").Call(Qual("time", "Now").Call(), Qual(packagenames.Ops, op), Lit(name))
}
