package lognoendwithperiod

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "lognoendwithperiod",
	Doc:      "check for log messages ending with period",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var names = set.NewFrozenStringSet(
	"(github.com/stackrox/rox/pkg/logging.Logger).Debug",
	"(github.com/stackrox/rox/pkg/logging.Logger).Debugf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Debugw",
	"(github.com/stackrox/rox/pkg/logging.Logger).Error",
	"(github.com/stackrox/rox/pkg/logging.Logger).Errorf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Errorw",
	"(github.com/stackrox/rox/pkg/logging.Logger).Fatal",
	"(github.com/stackrox/rox/pkg/logging.Logger).Fatalf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Fatalw",
	"(github.com/stackrox/rox/pkg/logging.Logger).Info",
	"(github.com/stackrox/rox/pkg/logging.Logger).Infof",
	"(github.com/stackrox/rox/pkg/logging.Logger).Infow",
	"(github.com/stackrox/rox/pkg/logging.Logger).Log",
	"(github.com/stackrox/rox/pkg/logging.Logger).Logf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Panic",
	"(github.com/stackrox/rox/pkg/logging.Logger).Panicf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Panicw",
	"(github.com/stackrox/rox/pkg/logging.Logger).Warn",
	"(github.com/stackrox/rox/pkg/logging.Logger).Warnf",
	"(github.com/stackrox/rox/pkg/logging.Logger).Warnw",
	"(github.com/stackrox/rox/pkg/logging.Logger).WriteToStderr",
)

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	common.FilteredPreorder(inspectResult, common.Not(common.IsTestFile), nodeFilter, func(n ast.Node) {
		checkCall(n.(*ast.CallExpr), pass)
	})
	return nil, nil
}

func checkCall(call *ast.CallExpr, pass *analysis.Pass) {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok {
		return
	}

	name := fn.FullName()
	if !names.Contains(name) {
		return
	}

	msg := call.Args[0]
	val := pass.TypesInfo.Types[msg].Value
	if val == nil {
		return
	}
	if val.Kind() != constant.String {
		return
	}

	s := constant.StringVal(val)
	if len(s) < 1 {
		return
	}

	if strings.HasSuffix(s, "...") {
		return
	}

	switch s[len(s)-1] {
	case '.', '!', '\n':
		pass.Report(analysis.Diagnostic{
			Pos:     msg.End() - 1,
			Message: fmt.Sprintf("Log message should not end with punctuation nor newlines: %q", s),
			SuggestedFixes: []analysis.SuggestedFix{{
				Message: "Remove trailing punctuation or newline",
				TextEdits: []analysis.TextEdit{
					{
						Pos:     msg.Pos(),
						End:     msg.End() - 1,
						NewText: []byte(s[:len(s)-1]),
					},
				},
			}},
		})
	}
}
