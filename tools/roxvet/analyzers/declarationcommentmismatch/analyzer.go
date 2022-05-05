package declarationcommentmismatch

import (
	"fmt"
	"go/ast"
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

		return true
	})
	return nil, nil
}

func checkFunction(funcType *ast.FuncDecl, pass *analysis.Pass) {
	funcName := funcType.Name.String()
	firstFuncLetter := []rune(funcName[0:1])[0]

	documentation := funcType.Doc
	if documentation != nil {
		commentSplit := strings.Split(documentation.List[0].Text, " ")
		if len(commentSplit) < 2 {
			return
		}

		firstCommentWord := commentSplit[1]
		firstCommentLetter := []rune(firstCommentWord[0:1])[0]

		if firstCommentWord[1:] == funcName[1:] && ((unicode.IsLower(firstFuncLetter) && unicode.ToUpper(firstFuncLetter) == firstCommentLetter) || (unicode.IsLower(firstCommentLetter) && unicode.ToUpper(firstCommentLetter) == firstFuncLetter)) {
			pass.Report(analysis.Diagnostic{
				Pos:     funcType.Pos(),
				Message: fmt.Sprintf("Mismatching comment/function capitalization for function %s and comment starting with %s", funcName, firstCommentWord),
			})
		}
	}
}
