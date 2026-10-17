package authfuncoverride

import (
	"go/ast"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a")
}

// TestAnalyzerWithPackageLevelVerifier demonstrates NewAnalyzer with a verifier that
// requires the authorizer to be a package-level variable.
func TestAnalyzerWithPackageLevelVerifier(t *testing.T) {
	testdata := analysistest.TestData()
	pkgLevelOnly := NewAnalyzer(func(expr ast.Expr, pass *analysis.Pass) bool {
		ident, ok := expr.(*ast.Ident)
		if !ok {
			return false
		}
		obj, ok := pass.TypesInfo.Uses[ident].(*types.Var)
		if !ok {
			return false
		}
		return obj.Parent() != nil && obj.Parent() == obj.Pkg().Scope()
	})
	analysistest.Run(t, testdata, pkgLevelOnly, "b")
}

func TestUserAuthorizerVerifier(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, NewAnalyzer(UserAuthorizerVerifier), "c")
}
