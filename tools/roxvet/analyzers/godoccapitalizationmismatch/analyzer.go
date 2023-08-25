package godoccapitalizationmismatch

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"

	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const doc = `check the capitalization of the first word for a Godoc comment (for a function, interface, struct)`

// Analyzer is a analysis.Analyzer from the analysis package of the Go standard lib. [It analyzes code]
var Analyzer = &analysis.Analyzer{
	Name:     "godoccapitalizationmismatch",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.ValueSpec)(nil),
		(*ast.GenDecl)(nil),
	}

	common.FilteredPreorder(inspectResult, common.Not(common.IsTestFile), nodeFilter, func(n ast.Node) {
		switch t := n.(type) {
		case *ast.FuncDecl:
			if t.Doc == nil {
				return
			}
			checkCommentCaseMatches(pass, t.Doc, t.Name.String(), "function", t.Doc.Pos())

		case *ast.ValueSpec:
			if t.Doc == nil {
				return
			}
			checkCommentCaseMatches(pass, t.Doc, t.Names[0].String(), "variable", t.Doc.Pos())

		case *ast.GenDecl:
			documentation := t.Doc
			if documentation == nil {
				return
			}
			for _, spec := range t.Specs {
				if astType, ok2 := spec.(*ast.TypeSpec); ok2 {
					switch astType.Type.(type) {
					case *ast.ArrayType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "array", documentation.Pos())
					case *ast.StructType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "struct", documentation.Pos())
					case *ast.InterfaceType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "interface", documentation.Pos())
					case *ast.MapType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "map", documentation.Pos())
					case *ast.ChanType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "channel", documentation.Pos())
					}
					return
				}
			}
		}
	})
	return nil, nil
}

func checkCommentCaseMatches(pass *analysis.Pass, doc *ast.CommentGroup, objectName string, objectType string, position token.Pos) {
	if len(doc.List) < 1 {
		return
	}
	commentSplit := strings.Fields(doc.List[0].Text)
	if len(commentSplit) < 2 || len(objectName) < 1 {
		return
	}

	firstObjectLetter := []rune(objectName[0:1])[0]
	firstCommentWord := commentSplit[1]
	if len(firstCommentWord) < 1 {
		return
	}
	firstCommentLetter := []rune(firstCommentWord[0:1])[0]

	if firstCommentWord[1:] == objectName[1:] {
		if unicode.IsLower(firstObjectLetter) && unicode.ToUpper(firstObjectLetter) == firstCommentLetter {
			pass.Report(analysis.Diagnostic{
				Pos:     position,
				Message: fmt.Sprintf("If a Godoc comment starts with a %s name, the capitalization of the first word in the comment must match the capitalization used in the %s name.\nChange '// %s' to '// %s%s'.", objectType, objectType, firstCommentWord, string(unicode.ToLower(firstCommentLetter)), firstCommentWord[1:]),
			})
		}
	}

}
