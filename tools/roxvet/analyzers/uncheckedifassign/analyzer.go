package uncheckedifassign

import (
	"go/ast"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `check for if conditions not depending on an inline assignment`

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "uncheckedifassign",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func getAssignedObjects(ifStmt *ast.IfStmt) map[*ast.Object]struct{} {
	objs := make(map[*ast.Object]struct{})
	switch initAssign := ifStmt.Init.(type) {
	case *ast.AssignStmt:
		for _, expr := range initAssign.Lhs {
			idExpr, _ := expr.(*ast.Ident)
			if idExpr == nil || idExpr.Obj == nil || idExpr.Obj.Kind != ast.Var {
				continue
			}
			objs[idExpr.Obj] = struct{}{}
		}
	case *ast.IncDecStmt:
		idExpr, _ := initAssign.X.(*ast.Ident)
		if idExpr != nil && idExpr.Obj != nil && idExpr.Obj.Kind == ast.Var {
			objs[idExpr.Obj] = struct{}{}
		}
	}
	return objs
}

func checkAnyObjectUsedIn(expr ast.Expr, objs map[*ast.Object]struct{}) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if found {
			return false
		}

		if ident, _ := node.(*ast.Ident); ident != nil {
			if _, ok := objs[ident.Obj]; ok {
				found = true
				return false
			}
		}

		return true
	})
	return found
}

func run(pass *analysis.Pass) (interface{}, error) {
	nodeFilter := []ast.Node{
		(*ast.IfStmt)(nil),
	}

	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	common.FilteredPreorder(inspectResult, common.Not(common.IsGeneratedFile), nodeFilter, func(n ast.Node) {
		ifStmt := n.(*ast.IfStmt)
		if ifStmt.Init == nil {
			return
		}

		assignedObjs := getAssignedObjects(ifStmt)
		if !checkAnyObjectUsedIn(ifStmt.Cond, assignedObjs) {
			pass.Reportf(ifStmt.Pos(), "condition in if statement does not depend on inline assignment")
		}
	})

	return nil, nil
}
