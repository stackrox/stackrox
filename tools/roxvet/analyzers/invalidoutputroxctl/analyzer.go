package invalidoutputroxctl

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	roxctlPrefixPath = "github.com/stackrox/rox/roxctl"
	testFileSuffix   = "_test.go"
)

var (
	disallowedFunctions = set.NewFrozenStringSet("fmt.Printf", "fmt.Print", "fmt.Println")

	disallowedReferences = set.NewFrozenStringSet("os.Stdout", "os.Stderr")

	// Some packages rely on os.Stdout/os.Stderr such as the environment package. Add these packages to ignoredPackages
	ignoredPackages = set.NewStringSet(roxctlPrefixPath + "/common/environment")
	// For now, need refactoredPackages to only lint for commands which have been refactored to environment usage.
	// If you are refactoring, make sure you add the package here. When done with refactoring, this can be removed.
	refactoredPackages = set.NewStringSet(roxctlPrefixPath + "/image/check")
)

// Analyzer provides the analyzer to check invalid output within roxctl. Invalid output either uses printing functions
// without giving an explicit output stream or redirects output to os.Stdout / os.Stderr
var Analyzer = &analysis.Analyzer{
	Name:     "invalidoutputroxctl",
	Doc:      "check whether env.IO().In/Out/ErrOut is used in roxctl commands instead of os.Stdin/Stdout/StdErr as well as printing functions without explicit writers",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// checkForPkgPathPrefixInSet checks whether the given pkgPath has a prefix within the given set.
func checkForPkgPathPrefixInSet(pkgPath string, set set.StringSet) bool {
	for pkgPathPrefix := range set {
		if strings.HasPrefix(pkgPath, pkgPathPrefix) {
			return true
		}
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	// Ignore packages that are either not under "roxctl", explicitly ignored or are not yet refactored
	if path := pass.Pkg.Path(); !strings.HasPrefix(path, roxctlPrefixPath) || checkForPkgPathPrefixInSet(path, ignoredPackages) || !checkForPkgPathPrefixInSet(path, refactoredPackages) {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.File)(nil),
		(*ast.CallExpr)(nil),
		(*ast.SelectorExpr)(nil),
		(*ast.Ident)(nil),
	}
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		if !push {
			return false
		}
		switch n := n.(type) {
		case *ast.File:
			// Ignore test files and ignored files for now, as test files may or may not depend on os / fmt functions
			// and ignored files are i.e. legacy code
			fileName := pass.Fset.File(n.Pos()).Name()
			if strings.HasSuffix(fileName, testFileSuffix) {
				return false
			}
		case *ast.CallExpr:
			fn, ok := typeutil.Callee(pass.TypesInfo, n).(*types.Func)
			if ok && disallowedFunctions.Contains(fn.FullName()) {
				pass.Reportf(n.Pos(), "Disallowed function used %q. Use"+
					" environment's functions for printing or to a specific writer from environment.InputOutput()", fn.FullName())
			}
		case *ast.SelectorExpr:
			verifySelectorExpression(n, pass)
		case *ast.Ident:
			verifyIdent(n, pass)
		}
		return true
	})
	return nil, nil
}

func verifyIdent(ident *ast.Ident, pass *analysis.Pass) {
	// When expression is of type *ast.Ident, we could potentially be dealing with a dot import of
	// packages and the identifier is a reference to an object within an imported package.
	// With the use of *types.Info we can check whether the identifier is associated with a package
	// or just declared within the package itself and use the pkg path and verify based on that
	reference := ident.Name
	if _, exists := pass.TypesInfo.Uses[ident]; exists && pass.TypesInfo.Uses[ident].Pkg() != nil {
		pkgPath := pass.TypesInfo.Uses[ident].Pkg().Path()
		qualifiedName := fmt.Sprintf("%s.%s", pkgPath, reference)
		if disallowedReferences.Contains(qualifiedName) {
			pass.Reportf(ident.Pos(), "Disallowed output streams used: %s.%s. Use "+
				"environment.InputOutput().In/Out instead.", pkgPath, reference)
		}
	}
}

func verifySelectorExpression(selectorExpr *ast.SelectorExpr, pass *analysis.Pass) {
	// When expression is of type *ast.SelectorExpr we could potentially be dealing with named imports
	// The import can be either the default pkg name:
	//   			import "os"
	// or a custom import alias:
	//				import myos "os"
	//
	// With the use of *types.Info we can filter out the actual imported package and
	// get the unmodified pkg path and name and verify based on that
	ident, ok := selectorExpr.X.(*ast.Ident)
	if !ok {
		return
	}
	reference := selectorExpr.Sel.Name
	obj, exists := pass.TypesInfo.Uses[ident]
	if !exists {
		return
	}
	pkgName, ok := obj.(*types.PkgName)
	if !ok {
		return
	}
	pkgPath := pkgName.Imported().Path()
	qualifiedName := fmt.Sprintf("%s.%s", pkgPath, reference)
	if disallowedReferences.Contains(qualifiedName) {
		pass.Reportf(selectorExpr.Pos(), "Disallowed output streams used: %s.%s. Use "+
			"environment.InputOutput().In/Out instead.", pkgPath, reference)
	}
}
