package operations

import (
	. "github.com/dave/jennifer/jen"
)

func incrementTxnCount() *Statement {
	return bIndex().Dot("IncTxnCount").Call()
}

func generateSetTxn(_ GeneratorProperties) (Code, Code) {
	sig := Id("SetTxnCount").Params(Id("seq").Uint64()).Error()
	impl := renderFuncBStarIndexer().Add(sig).Block(
		Return().Add(bIndex().Dot("SetTxnCount").Call(Id("seq"))),
	)
	return sig, impl
}

func generateGetTxn(_ GeneratorProperties) (Code, Code) {
	sig := Id("GetTxnCount").Params().Parens(List(Uint64()))
	impl := renderFuncBStarIndexer().Add(sig).Block(
		Return().Add(bIndex().Dot("GetTxnCount").Call()),
	)
	return sig, impl
}

func init() {
	supportedMethods["set_txn"] = generateSetTxn
	supportedMethods["get_txn"] = generateGetTxn

}
