package declarationcommentmismatch

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

const doc = `check that there are no functions, interfaces or structs preceded by descriptive comments of mismatching capitalization`

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "declarationcommentmismatch",
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
	}

	inspectResult.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		if !push {
			return true
		}

		if astFile, ok := n.(*ast.File); ok {
			file := pass.Fset.File(astFile.Pos())
			if file != nil && strings.HasSuffix(file.Name(), "_test.go") {
				return false
			}
			return true
		}

		if astFunction, ok := n.(*ast.FuncDecl); ok {
			checkFunction(astFunction, pass)
			return true
		}

		if astVar, ok := n.(*ast.ValueSpec); ok {
			checkVariable(astVar, pass)
			return true
		}

		return true
	})
	return nil, nil
}

func checkFunction(funcType *ast.FuncDecl, pass *analysis.Pass) {
	checkDocumentation(pass, funcType.Doc, funcType.Name.String(), "function", funcType.Pos())
}

func checkVariable(valueSpec *ast.ValueSpec, pass *analysis.Pass) {
	checkDocumentation(pass, valueSpec.Doc, valueSpec.Names[0].String(), "variable", valueSpec.Pos())
}

func checkDocumentation(pass *analysis.Pass, doc *ast.CommentGroup, objectName string, objectType string, position token.Pos) {
	if doc == nil {
		return
	}
	commentSplit := strings.Split(doc.List[0].Text, " ")
	if len(commentSplit) < 2 {
		return
	}

	firstObjectLetter := []rune(objectName[0:1])[0]

	firstCommentWord := commentSplit[1]
	firstCommentLetter := []rune(firstCommentWord[0:1])[0]

	if firstCommentWord[1:] == objectName[1:] && ((unicode.IsLower(firstObjectLetter) && unicode.ToUpper(firstObjectLetter) == firstCommentLetter) || (unicode.IsLower(firstCommentLetter) && unicode.ToUpper(firstCommentLetter) == firstObjectLetter)) {
		pass.Report(analysis.Diagnostic{
			Pos:     position,
			Message: fmt.Sprintf("Mismatching capitalization for %s %s and comment starting with %s", objectType, objectName, firstCommentWord),
		})
	}
}
