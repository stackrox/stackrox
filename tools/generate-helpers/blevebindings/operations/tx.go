package operations

import (
	. "github.com/dave/jennifer/jen"
)

func incrementTxnCount(needsTxManager bool) *Statement {
	if needsTxManager {
		return bIndex().Dot("IncTxnCount").Call()
	}
	return Nil()
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
	supportedTxnMethods["set_txn"] = generateSetTxn
	supportedTxnMethods["get_txn"] = generateGetTxn
}
