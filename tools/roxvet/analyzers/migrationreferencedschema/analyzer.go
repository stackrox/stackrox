package migrationreferencedschema

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	doc = `check for usages of ResolveReferences`

	resolveReferences = `ResolveReferences`
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "migrationreferencedschemas",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	fmt.Printf("SHREWS -- package %q\n", pass.Pkg.Path())
	//root, valid, err := getRoot(pass.Pkg.Path())
	//if err != nil {
	//	pass.Reportf(token.NoPos, "couldn't find valid root: %v", err)
	//	return nil, nil
	//}
	//if !valid {
	//	return nil, nil
	//}
	//
	//// If we are not the migrator, simply move along
	//if !strings.HasPrefix(root, "migrator") {
	//	return nil, nil
	//} else {
	//	// This is allowed in frozen schema or the initial frozen schema as that code is isolaged
	//	if strings.Contains(root, "postgreshelper") || strings.Contains(root, "frozenschema") {
	//		return nil, nil
	//	}
	//}

	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if ok && fn.Name() == resolveReferences {
			fmt.Printf("function full name is: %q\n", fn.FullName())
			fmt.Printf("function package is: %q\n", fn.Pkg())
			pass.Report(analysis.Diagnostic{
				Pos:     n.Pos(),
				Message: "Cannot resolve references in a migration as it walks the most recent proto.",
			})
		}
	})
	return nil, nil
}
