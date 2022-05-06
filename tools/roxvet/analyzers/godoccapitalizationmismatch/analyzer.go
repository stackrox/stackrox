package godoccapitalizationmismatch

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"
	"unicode"

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
		(*ast.File)(nil),
		(*ast.FuncDecl)(nil),
		(*ast.ValueSpec)(nil),
		(*ast.GenDecl)(nil),
	}

	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		if astFile, ok := n.(*ast.File); ok {
			file := pass.Fset.File(astFile.Pos())
			return file == nil || !strings.HasSuffix(file.Name(), "_test.go")
		}

		if astFunction, ok := n.(*ast.FuncDecl); ok {
			if astFunction.Doc == nil {
				return true
			}
			checkCommentCaseMatches(pass, astFunction.Doc, astFunction.Name.String(), "function", astFunction.Doc.Pos())
			return true
		}

		if astVar, ok := n.(*ast.ValueSpec); ok {
			if astVar.Doc == nil {
				return true
			}
			checkCommentCaseMatches(pass, astVar.Doc, astVar.Names[0].String(), "variable", astVar.Doc.Pos())
			return true
		}

		if astGenDecl, ok := n.(*ast.GenDecl); ok {
			documentation := astGenDecl.Doc
			if documentation == nil {
				return true
			}
			for _, spec := range astGenDecl.Specs {
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
					return true
				}
			}
		}

		return true
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
