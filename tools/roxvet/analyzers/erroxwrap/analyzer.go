package erroxwrap

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `Detect improper wrapping of errox errors with errors.Wrap/Wrapf

errox errors should use their own New/Newf or CausedBy/CausedByf methods
instead of being wrapped with pkg/errors.Wrap or errors.Wrapf.

Bad:
  errors.Wrap(errox.InvalidArgs, "message")
  errors.Wrapf(errox.NotFound, "message %s", arg)

Good:
  errox.InvalidArgs.New("message")
  errox.NotFound.Newf("message %s", arg)
  errox.InvalidArgs.CausedBy(err)
`

var Analyzer = &analysis.Analyzer{
	Name:     "erroxwrap",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Known errox sentinel errors from pkg/errox
var erroxSentinels = map[string]bool{
	"github.com/stackrox/rox/pkg/errox.NotFound":                  true,
	"github.com/stackrox/rox/pkg/errox.AlreadyExists":             true,
	"github.com/stackrox/rox/pkg/errox.InvalidArgs":               true,
	"github.com/stackrox/rox/pkg/errox.ReferencedByAnotherObject": true,
	"github.com/stackrox/rox/pkg/errox.InvariantViolation":        true,
	"github.com/stackrox/rox/pkg/errox.NoCredentials":             true,
	"github.com/stackrox/rox/pkg/errox.NotAuthorized":             true,
	"github.com/stackrox/rox/pkg/errox.NoAuthzConfigured":         true,
	"github.com/stackrox/rox/pkg/errox.ServerError":               true,
	"github.com/stackrox/rox/pkg/errox.ResourceExhausted":         true,
	"github.com/stackrox/rox/pkg/errox.NotImplemented":            true,
	"github.com/stackrox/rox/pkg/errox.ReferencedObjectNotFound":  true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok {
			return
		}

		fullName := fn.FullName()

		// Check if this is errors.Wrap or errors.Wrapf
		if fullName != "github.com/pkg/errors.Wrap" && fullName != "github.com/pkg/errors.Wrapf" {
			return
		}

		// Check if the first argument is an errox sentinel error
		if len(call.Args) < 2 {
			return
		}

		firstArg := call.Args[0]
		if isErroxSentinel(pass.TypesInfo, firstArg) {
			erroxName := getErroxName(firstArg)
			isWrapf := strings.HasSuffix(fullName, ".Wrapf")

			// Check if any of the remaining arguments could be an error
			hasErrorArg := false
			for i := 2; i < len(call.Args); i++ {
				if isErrorType(pass.TypesInfo, call.Args[i]) {
					hasErrorArg = true
					break
				}
			}

			// Build the replacement code
			var methodName string
			if hasErrorArg {
				// An error is being wrapped - use CausedBy or CausedByf
				if isWrapf {
					methodName = "CausedByf"
				} else {
					methodName = "CausedBy"
				}
			} else {
				// No error being wrapped - use New or Newf
				if isWrapf {
					methodName = "Newf"
				} else {
					methodName = "New"
				}
			}

			// Create the replacement call expression
			newCall := buildReplacementCall(firstArg, methodName, call.Args[1:])
			replacementText := formatNode(pass.Fset, newCall)

			message := "Use " + erroxName + "." + methodName + " instead of errors." +
				strings.TrimPrefix(fn.Name(), "errors.") + ": " + replacementText

			pass.Report(analysis.Diagnostic{
				Pos:     call.Pos(),
				End:     call.End(),
				Message: message,
				SuggestedFixes: []analysis.SuggestedFix{
					{
						Message: "Replace with " + erroxName + "." + methodName,
						TextEdits: []analysis.TextEdit{
							{
								Pos:     call.Pos(),
								End:     call.End(),
								NewText: []byte(replacementText),
							},
						},
					},
				},
			})
		}
	})

	return nil, nil
}

// isErroxSentinel checks if the expression is an errox sentinel error
func isErroxSentinel(info *types.Info, expr ast.Expr) bool {
	obj := info.ObjectOf(identOf(expr))
	if obj == nil {
		return false
	}

	// Get the fully qualified name
	if pkg := obj.Pkg(); pkg != nil {
		fullName := pkg.Path() + "." + obj.Name()
		return erroxSentinels[fullName]
	}
	return false
}

// identOf extracts the identifier from an expression (handling selectors like errox.NotFound)
func identOf(expr ast.Expr) *ast.Ident {
	switch e := expr.(type) {
	case *ast.Ident:
		return e
	case *ast.SelectorExpr:
		return e.Sel
	default:
		return nil
	}
}

// getErroxName extracts a friendly name for the errox error for use in messages
func getErroxName(expr ast.Expr) string {
	if sel, ok := expr.(*ast.SelectorExpr); ok {
		if ident, ok := sel.X.(*ast.Ident); ok {
			return ident.Name + "." + sel.Sel.Name
		}
	}
	return "errox error"
}

// isErrorType checks if the expression has type error or implements error interface
func isErrorType(info *types.Info, expr ast.Expr) bool {
	t := info.TypeOf(expr)
	if t == nil {
		return false
	}

	// Check if the type implements error interface
	errorType := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return types.Implements(t, errorType) || types.Implements(types.NewPointer(t), errorType)
}

// buildReplacementCall creates a new call expression like errox.NotFound.Newf(...)
func buildReplacementCall(erroxExpr ast.Expr, methodName string, args []ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{
		Fun: &ast.SelectorExpr{
			X:   erroxExpr,
			Sel: ast.NewIdent(methodName),
		},
		Args: args,
	}
}

// formatNode formats an AST node as a string
func formatNode(fset *token.FileSet, node ast.Node) string {
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, node); err != nil {
		return ""
	}
	return buf.String()
}
