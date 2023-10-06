package structuredlogs

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "structuredlogs",
	Doc:      "check for structured logs usage and flag non-structured logs",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var replacements = map[string]map[string]string{
	"github.com/stackrox/rox/pkg/logging": {
		"Logger.Warnf":  "Warnw",
		"Logger.Errorf": "Errorw",
		"Logger.Fatalf": "Fatalw",
	},
}

// List of the linted package paths. Since structured logs aren't being rolled out immediately to all packages,
// we will gradually increase the list here. Ultimately, all logs should be moved to structured logs.
var packagesToLint = []*regexp.Regexp{
	regexp.MustCompile(`^github\.com/stackrox/rox/central/reprocessor(/|$)|^github\.com/stackrox/rox/central/image/service(/|$)|^github\.com/stackrox/rox/pkg/notifiers(/|$)`),
}

func run(pass *analysis.Pass) (interface{}, error) {
	if !matchesPackagePattern(pass.Pkg.Path()) {
		return nil, nil
	}

	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	fileFilter := common.And(common.Not(common.IsTestFile), common.Not(common.IsGeneratedFile))

	common.FilteredPreorder(inspectResult, fileFilter, nodeFilter, func(n ast.Node) {
		checkCall(pass, n.(*ast.CallExpr))
	})

	return nil, nil
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	fn := astutil.Unparen(call.Fun)

	// Skip type conversions and built-in functions.
	if pass.TypesInfo.Types[fn].IsType() || pass.TypesInfo.Types[fn].IsBuiltin() {
		return
	}

	namedFn, _ := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if namedFn == nil {
		return
	}

	if match, name, replacement := isNonStructuredLogFunction(namedFn); match {
		pass.Reportf(call.Fun.Pos(), "Logging function %s used without structured context; use %s instead",
			name, replacement)
	}
}

func isNonStructuredLogFunction(fn *types.Func) (bool, string, string) {
	sig := fn.Type().(*types.Signature)
	if sig == nil {
		return false, "", ""
	}

	if fn.Pkg() == nil {
		return false, "", ""
	}
	logReplacements, isLogFunction := replacements[fn.Pkg().Path()]
	if !isLogFunction {
		return false, "", ""
	}

	name := fn.Name()

	if sig.Recv() != nil {
		receiverType := sig.Recv().Type()
		qualifier := types.RelativeTo(fn.Pkg())
		name = fmt.Sprintf("%s.%s", types.TypeString(receiverType, qualifier), name)
	}
	replacement, match := logReplacements[name]
	return match, fmt.Sprintf("\"%s\".%s", fn.Pkg().Path(), name), replacement
}

func matchesPackagePattern(path string) bool {
	for _, regex := range packagesToLint {
		if regex.MatchString(path) {
			return true
		}
	}
	return false
}
