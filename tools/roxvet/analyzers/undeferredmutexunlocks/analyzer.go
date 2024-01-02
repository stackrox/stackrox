package undeferredmutexunlocks

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/tools/roxvet/common"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

////////////////////////////////////////////////////////////////////////////////
//
// Undeferred explicit mutex unlock is not a great pattern because the control
// might not reach the Unlock() call and leave the mutex in the locked state.
// Instead, either use:
//   * defer to ensure Unlock() is called upon function exit
//   * or concurrency.With[R]Lock[N]() if you like to unlock earlier.
//
// In some cases calling Unlock() explicitly is safe, for example:
// ```
//	g.m.Lock()
//	g.opts = opts
//	g.m.Unlock()
// ```
// However, for consistency and simplicity we want to require With[R]Lock[N]()
// in such cases as well:
// ```
//	concurrency.WithLock(&g.m, func() {
//		g.opts = opts
//	})
// ```
//
// Sometimes (rarely) you need an explicit Unlock() call, for example:
// ```
//	mutex.Lock()
//	if c, ok := g.m[key]; ok {
//		doSyncCall1()
//		mutex.Unlock()
//
//		doCall1()
//	}
//	doSyncCall2()
//	mutex.Unlock()
//
//	doCall2()
// ```
// In such cases use Unsafe[R]Unlock() to suppress the error:
// ```
//	concurrency.UnsafeUnlock(&mutex)
// ```
//
////////////////////////////////////////////////////////////////////////////////

const doc = `check for mutex [R]Unlock() calls that are not deferred`

// Analyzer is the analyzer.
var Analyzer = &analysis.Analyzer{
	Name:     "undeferredmutexunlocks",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// A list of targets for this analyzer, i.e., functions which should not be
// called undeferred. For now, we don't care about fully qualified type names;
// pointer, non-pointer, or no receivers.
var targets = set.NewFrozenStringSet(
	"Mutex.Unlock",
	"RWMutex.Unlock",
	"RWMutex.RUnlock",
	"KeyedMutex.Unlock",
	"KeyedRWMutex.Unlock",
	"KeyedRWMutex.RUnlock",
	"KeyFence.Unlock",
)

func isTargetFun(fun *types.Func) bool {
	if fun == nil {
		return false
	}
	if sig := fun.Type().(*types.Signature); sig != nil && sig.Recv() != nil {
		recvTyp := types.TypeString(sig.Recv().Type(), types.RelativeTo(fun.Pkg()))
		recvTyp, _ = stringutils.MaybeTrimPrefix(recvTyp, "*")

		formatted := fmt.Sprintf("%s.%s", recvTyp, fun.Name())
		if targets.Contains(formatted) {
			return true
		}
	}
	return false
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Keeps positions for every deferred expression so when looking at a
	// function call we can check if it is called under defer or not.
	deferredExpressions := set.NewSet[token.Pos]()

	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
		(*ast.DeferStmt)(nil),
	}

	// There seems to be no need to exclude auto-generated files at the moment
	// of writing so keeping it simple for now.
	fileFilter := common.Not(common.IsTestFile)

	// Preorder guarantees that we see defer statement before its underlying
	// expression.
	common.FilteredPreorder(inspectResult, fileFilter, nodeFilter, func(n ast.Node) {
		switch n := n.(type) {
		case *ast.DeferStmt:
			deferredExpressions.Add(n.Call.Pos())
		case *ast.CallExpr:
			// Skipping type conversions and builtins does not seem to improve
			// analyzer runtime.

			fun, _ := typeutil.Callee(pass.TypesInfo, n).(*types.Func)
			if fun != nil && isTargetFun(fun) && !deferredExpressions.Contains(n.Pos()) {
				pass.Reportf(n.Pos(), "do not unlock a mutex without defer, use With[R]Lock[N]() pattern;"+
					" to disable, use Unsafe[R]Unlock()")
			}
		default:
			panic(fmt.Sprintf("Unexpected type: %T", n))
		}
	})

	return nil, nil
}
