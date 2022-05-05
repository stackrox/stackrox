package commentcapitalization

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

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "commentcapitalization",
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
		(*ast.TypeSpec)(nil),
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
			checkCommentCaseMatches(pass, astFunction.Doc, astFunction.Name.String(), "function", astFunction.Pos())
			return true
		}

		if astVar, ok := n.(*ast.ValueSpec); ok {
			checkCommentCaseMatches(pass, astVar.Doc, astVar.Names[0].String(), "variable", astVar.Pos())
			return true
		}

		if astGenDecl, ok := n.(*ast.GenDecl); ok {
			documentation := astGenDecl.Doc
			for _, spec := range astGenDecl.Specs {
				if astType, ok2 := spec.(*ast.TypeSpec); ok2 {
					switch astType.Type.(type) {
					case *ast.ArrayType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "array", astType.Pos())
					case *ast.StructType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "struct", astType.Pos())
					case *ast.InterfaceType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "interface", astType.Pos())
					case *ast.MapType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "map", astType.Pos())
					case *ast.ChanType:
						checkCommentCaseMatches(pass, documentation, astType.Name.String(), "channel", astType.Pos())
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
	if len(commentSplit) < 2 {
		return
	}
	if len(objectName) < 1 {
		return
	}

	firstObjectLetter := []rune(objectName[0:1])[0]
	firstCommentWord := commentSplit[1]
	if len(firstCommentWord) < 1 {
		return
	}
	firstCommentLetter := []rune(firstCommentWord[0:1])[0]

	if firstCommentWord[1:] == objectName[1:] && ((unicode.IsLower(firstObjectLetter) && unicode.ToUpper(firstObjectLetter) == firstCommentLetter) || (unicode.IsLower(firstCommentLetter) && unicode.ToUpper(firstCommentLetter) == firstObjectLetter)) {
		pass.Report(analysis.Diagnostic{
			Pos:     position,
			Message: fmt.Sprintf("Mismatching capitalization for %s %s and comment starting with %s", objectType, objectName, firstCommentWord),
		})
	}
}
