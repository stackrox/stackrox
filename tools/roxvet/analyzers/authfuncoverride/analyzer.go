package authfuncoverride

import (
	"go/ast"
	"go/types"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `check that AuthFuncOverride methods conform to the expected pattern

Every method named AuthFuncOverride must:
  - have a pointer receiver
  - have signature: func (*T) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)
  - have body consisting of exactly one statement: return ctx, <authorizer>.Authorized(ctx, fullMethodName)
  - use an authorizer that satisfies the configured verifier (when one is provided)`

// Analyzer checks AuthFuncOverride methods. It accepts any authorizer without additional
// constraints; use NewAnalyzer to add authorizer constraints via a custom verifier.
var Analyzer = NewAnalyzer(nil)

// AuthorizerVerifier inspects the authorizer expression used inside an AuthFuncOverride
// body. expr is the AST expression of the authorizer (the receiver of the .Authorized
// call). Returning false causes the method to be flagged.
type AuthorizerVerifier func(authorizerExpr ast.Expr, pass *analysis.Pass) bool

// NewAnalyzer returns an Analyzer that validates AuthFuncOverride methods.
// verifier is invoked with the authorizer expression and the analysis pass; if nil any
// authorizer is accepted. Returning false from the verifier causes a diagnostic.
func NewAnalyzer(verifier AuthorizerVerifier) *analysis.Analyzer {
	a := &authFuncOverrideAnalyzer{verifier: verifier}
	return &analysis.Analyzer{
		Name:     "authfuncoverride",
		Doc:      doc,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
		Run:      a.run,
	}
}

type authFuncOverrideAnalyzer struct {
	verifier AuthorizerVerifier
}

var nodeFilter = []ast.Node{(*ast.FuncDecl)(nil)}

func (a *authFuncOverrideAnalyzer) run(pass *analysis.Pass) (any, error) {
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	common.FilteredPreorder(insp, common.Not(ast.IsGenerated), nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)
		if fn.Name.Name != "AuthFuncOverride" || fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}

		if !checkSignature(pass, fn) {
			pass.Reportf(fn.Pos(), "AuthFuncOverride must have signature: func (*T) AuthFuncOverride(ctx context.Context, fullMethodName string) (context.Context, error)")
			return
		}

		authorizerExpr, ok := checkBody(pass, fn)
		if !ok {
			pass.Reportf(fn.Pos(), "AuthFuncOverride body must be exactly: return ctx, <authorizer>.Authorized(ctx, fullMethodName)")
			return
		}

		if a.verifier != nil && !a.verifier(authorizerExpr, pass) {
			pass.Reportf(fn.Pos(), "AuthFuncOverride authorizer does not satisfy the required constraints")
		}
	})
	return nil, nil
}

// checkSignature verifies the method has the required signature using type information.
func checkSignature(pass *analysis.Pass, fn *ast.FuncDecl) bool {
	defObj := pass.TypesInfo.Defs[fn.Name]
	if defObj == nil {
		return false
	}
	sig, ok := defObj.Type().(*types.Signature)
	if !ok {
		return false
	}

	recv := sig.Recv()
	if recv == nil {
		return false
	}
	if _, ok := recv.Type().(*types.Pointer); !ok {
		return false
	}

	params := sig.Params()
	if params.Len() != 2 {
		return false
	}
	if !isContextContext(params.At(0).Type()) {
		return false
	}
	if !isString(params.At(1).Type()) {
		return false
	}

	results := sig.Results()
	if results.Len() != 2 {
		return false
	}
	if !isContextContext(results.At(0).Type()) {
		return false
	}
	if !isError(results.At(1).Type()) {
		return false
	}

	return true
}

// checkBody verifies the body is exactly: return ctx, <authorizer>.Authorized(ctx, fullMethodName)
// It returns the authorizer expression (the receiver of Authorized) on success.
func checkBody(pass *analysis.Pass, fn *ast.FuncDecl) (authorizerExpr ast.Expr, ok bool) {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return nil, false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 2 {
		return nil, false
	}

	defObj := pass.TypesInfo.Defs[fn.Name]
	sig := defObj.Type().(*types.Signature)
	ctxParam := sig.Params().At(0)
	methodParam := sig.Params().At(1)

	// First return value must be the ctx parameter itself.
	ctxIdent, ok := ret.Results[0].(*ast.Ident)
	if !ok || pass.TypesInfo.Uses[ctxIdent] != ctxParam {
		return nil, false
	}

	// Second return value must be: <authorizer>.Authorized(ctx, fullMethodName).
	call, ok := ret.Results[1].(*ast.CallExpr)
	if !ok || len(call.Args) != 2 || call.Ellipsis.IsValid() {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Authorized" {
		return nil, false
	}

	arg0, ok := call.Args[0].(*ast.Ident)
	if !ok || pass.TypesInfo.Uses[arg0] != ctxParam {
		return nil, false
	}
	arg1, ok := call.Args[1].(*ast.Ident)
	if !ok || pass.TypesInfo.Uses[arg1] != methodParam {
		return nil, false
	}

	return sel.X, true
}

func isContextContext(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}

func isString(t types.Type) bool {
	b, ok := t.(*types.Basic)
	return ok && b.Kind() == types.String
}

func isError(t types.Type) bool {
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() == nil && obj.Name() == "error"
}
