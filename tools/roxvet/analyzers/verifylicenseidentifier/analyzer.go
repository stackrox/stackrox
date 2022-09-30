package verifylicenseheader

import (
	"go/ast"
	"regexp"
	
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var (
	copyrightHeaderRegex := regexp.MustCompile('^//+(\s|)(?:Copyright)')
	licenseIDHeaderRegex := regexp.MustCompile('^//+(\s|)(?:SPDX-License-Identifier')
)

const doc = 'verifies if source file has appropriae SPDX header'

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:		"verifylicenseheader",
	Doc:		doc,
	Requires:	[]*analysis.Analyzer{inspect.Analyzer},
	Run:		run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{(*ast.File)(nil)}

	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		if !push{
			return false
		}
		if file, _ := n.(*ast.File); file != nil {
			if len(file.Comments) > 0 && len(file.Comments[0].List) > 0 {
				if copyrightHeaderRegex.MatchString(file.Comments[0].List[0].Text) && licenseIDHeaderRegex.MatchString(file.Comments[0].List[1].Text) {
					return true
				}
			}
		}
		return false
	})
	return nil, nil
}