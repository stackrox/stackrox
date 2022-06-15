package unusedroxctlargs

import (
	"go/ast"
	"regexp"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	doc = `check for unused roxctl arguments`

	cobraCommandType = `github.com/spf13/cobra.Command`
	runE             = `RunE`
	wildcard         = `_`
)

var roxctlPkgPattern = regexp.MustCompile(`^github.com/stackrox/stackrox/roxctl(/|$)`)

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "unusedroxctlargs",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	if !roxctlPkgPattern.MatchString(pass.Pkg.Path()) {
		return nil, nil
	}

	nodeFilter := []ast.Node{(*ast.CompositeLit)(nil)}
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		compositeLitNode := n.(*ast.CompositeLit)
		if typ, ok := pass.TypesInfo.Types[compositeLitNode]; !ok || typ.Type.String() != cobraCommandType {
			return false
		}

		for _, elem := range compositeLitNode.Elts {
			kv, _ := elem.(*ast.KeyValueExpr)
			if kv == nil {
				continue
			}

			key, _ := kv.Key.(*ast.Ident)
			if key == nil || key.Name != runE {
				continue
			}

			fun, _ := kv.Value.(*ast.FuncLit)
			if fun == nil {
				continue
			}

			if fun.Type.Params.NumFields() != 2 {
				continue
			}
			argsNames := fun.Type.Params.List[1].Names
			if len(argsNames) == 0 || argsNames[0].Name == wildcard {
				// The args parameter is unnamed or a wildcard.
				pass.Reportf(n.Pos(), "RunE args argument is not used; the function should be wrapped in util.RunENoArgs")
			}
		}
		return false
	})

	return nil, nil
}
