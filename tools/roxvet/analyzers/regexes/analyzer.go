package regexes

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"regexp"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `Inspect usage for regexp functions for proper hygiene`

// Analyzer is the go vet entrypoint
var Analyzer = &analysis.Analyzer{
	Name:     "regexes",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	staticEvalFnMap = map[string]func(string) (*regexp.Regexp, error){
		"regexp.MustCompile":      regexp.Compile,
		"regexp.MustCompilePOSIX": regexp.CompilePOSIX,
	}

	forbidConstantArgFnMap = map[string]string{
		"regexp.Compile":      "regexp.MustCompile",
		"regexp.CompilePOSIX": "regexp.MustComilePOSIX",
		"regexp.Match":        "regexp.MustCompile",
		"regexp.MatchString":  "regexp.MustCompile",
		"regexp.MatchReader":  "regexp.MustCompile",
	}
)

func visitCall(call *ast.CallExpr, pass *analysis.Pass, topLevelScope bool) {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok {
		return
	}

	evalFn := staticEvalFnMap[fn.FullName()]
	if evalFn != nil {
		constVal, ok := stringConstantArg(pass, call, 0)
		if ok {
			_, err := evalFn(constVal)
			if err != nil {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Args[0].Pos(),
					Message: fmt.Sprintf("Invalid regex supplied to %s: %v", fn.FullName(), err),
				})
			}

			if topLevelScope {
				pass.Report(analysis.Diagnostic{
					Pos:     call.Pos(),
					Message: fmt.Sprintf("Function %s with a constant argument should be used at the top-level scope only", fn.FullName()),
				})
			}
		}
		return
	}

	replacement := forbidConstantArgFnMap[fn.FullName()]
	if replacement != "" {
		_, ok := stringConstantArg(pass, call, 0)
		if ok {
			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				Message: fmt.Sprintf("Do not use %s with a constant first argument, use %s at global scope instead", fn.FullName(), replacement),
			})
		}
		return
	}
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.CallExpr)(nil),
	}

	var topLevelFunc ast.Node
	common.FilteredNodes(inspectResult, common.Not(common.IsTestFile), nodeFilter, func(n ast.Node, push bool) bool {
		if !push {
			if topLevelFunc == n {
				topLevelFunc = nil
			}
			return true
		}

		if _, ok := n.(*ast.FuncDecl); ok {
			if topLevelFunc == nil {
				topLevelFunc = n
			}
			return true
		}

		visitCall(n.(*ast.CallExpr), pass, topLevelFunc != nil)
		return true
	})
	return nil, nil
}

func stringConstantArg(pass *analysis.Pass, call *ast.CallExpr, idx int) (string, bool) {
	if idx >= len(call.Args) {
		return "", false
	}
	arg := call.Args[idx]
	lit := pass.TypesInfo.Types[arg].Value
	if lit != nil && lit.Kind() == constant.String {
		return constant.StringVal(lit), true
	}
	return "", false
}
