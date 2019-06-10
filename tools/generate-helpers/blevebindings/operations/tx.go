package operations

import (
	"github.com/dave/jennifer/jen"
)

func incrementTxnCount() *jen.Statement {
	return bIndex().Dot("IncTxnCount").Call()
}

func generateSetTxn(_ GeneratorProperties) (jen.Code, jen.Code) {
	sig := jen.Id("SetTxnCount").Params(jen.Id("seq").Uint64()).Error()
	impl := renderFuncBStarIndexer().Add(sig).Block(
		jen.Return().Add(bIndex().Dot("SetTxnCount").Call(jen.Id("seq"))),
	)
	return sig, impl
}

func generateGetTxn(_ GeneratorProperties) (jen.Code, jen.Code) {
	sig := jen.Id("GetTxnCount").Params().Parens(jen.List(jen.Uint64()))
	impl := renderFuncBStarIndexer().Add(sig).Block(
		jen.Return().Add(bIndex().Dot("GetTxnCount").Call()),
	)
	return sig, impl
}

func init() {
	supportedMethods["set_txn"] = generateSetTxn
	supportedMethods["get_txn"] = generateGetTxn

}
