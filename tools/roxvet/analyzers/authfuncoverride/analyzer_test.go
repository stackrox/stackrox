package authfuncoverride

import (
	"go/ast"
	"go/types"
	"testing"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzerWithoutVerifier(t *testing.T) {
	testdata := analysistest.TestData()
	noVerifierAnalyzer := NewAnalyzer(nil)
	analysistest.Run(t, testdata, noVerifierAnalyzer, "a")
}

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "c")
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

// TestExcludedPkgPaths verifies that packages matching an excluded path prefix are
// skipped entirely. Package "d" contains a violation that would normally be flagged;
// excluding it must produce zero diagnostics (the file carries no // want markers, so
// any unexpected diagnostic would cause the test to fail).
func TestExcludedPkgPaths(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, NewAnalyzer(UserAuthorizerVerifier, "d"), "d")
}

// TestUserAuthorizerVerifierCrossPackage verifies that explicitly allow-listed
// cross-package variables (e.g. internal.Authorizer) are accepted, while other
// SelectorExpr forms (struct fields, untrusted packages) are rejected.
func TestUserAuthorizerVerifierCrossPackage(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, NewAnalyzer(UserAuthorizerVerifier), "github.com/stackrox/rox/central/externalbackups/service/svc")
}
