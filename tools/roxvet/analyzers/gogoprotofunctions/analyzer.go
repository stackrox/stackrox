package gogoprotofunctions

import (
	"fmt"
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	doc = `check for usages of "github.com/gogo/protobuf/proto" and "github.com/gogo/protobuf/types" functions`

	messageFormat = "Cannot call directly the %q function from the %q package. " +
		"Use the %q function from the \"github.com/stackrox/rox/pkg/protocompat\" package instead."
)

var (
	allowedCallerPackages = []string{
		"github.com/stackrox/rox/pkg/protocompat",
		"github.com/stackrox/rox/pkg/protoconv",
	}

	monitoredFunctions = map[string]map[string]string{
		// Key is the gogo protobuf package
		// value is a replaced/replacement function map where
		// - inner key is the function name in the gogo protobuf library
		// - inner value is the replacement function name in protocompat
		"github.com/gogo/protobuf/proto": {
			"Clone":             "Clone",
			"Marshal":           "Marshal",
			"MarshalTextString": "MarshalTextString",
			"Equal":             "Equal",
			"Unmarshal":         "Unmarshal",
		},
		"github.com/gogo/protobuf/types": {
			"Compare":            "CompareTimestamps",
			"DurationFromProto":  "DurationFromProto",
			"DurationProto":      "DurationProto",
			"TimestampFromProto": "ConvertTimestampToTimeOrError",
			"TimestampNow":       "TimestampNow",
			"TimestampProto":     "ConvertTimeToTimestampOrError",
		},
	}

	extraAllowedCallerPackages = map[string]map[string][]string{
		"github.com/gogo/protobuf/proto": {
			"Clone": {
				"github.com/stackrox/rox/pkg/protoutils",
			},
			"Unmarshal": {
				"github.com/stackrox/rox/pkg/postgres/pgutils",
				"github.com/stackrox/rox/pkg/search/postgres",
			},
		},
	}

	// Analyzer is the analyzer.
	Analyzer = &analysis.Analyzer{
		Name:     "gogoprotofunctions",
		Doc:      doc,
		Run:      run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
)

func run(pass *analysis.Pass) (interface{}, error) {
	callerPkg := pass.Pkg.Path()
	for _, allowedPkg := range allowedCallerPackages {
		if allowedPkg == callerPkg {
			return nil, nil
		}
	}

	nodeFilter := []ast.Node{(*ast.CallExpr)(nil)}
	inspectResult := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	inspectResult.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
		if !ok || fn.Pkg() == nil {
			return
		}
		for pkg, mapping := range monitoredFunctions {
			if fn.Pkg().Path() != pkg {
				continue
			}
			replacedPkgExtraAllowedCallers := extraAllowedCallerPackages[pkg]
			for monitoredFunction, replacement := range mapping {
				if fn.Name() != monitoredFunction {
					continue
				}
				extraAllowedPackages := replacedPkgExtraAllowedCallers[monitoredFunction]
				isCallerPkgAllowed := false
				for _, extraAllowedPkg := range extraAllowedPackages {
					if callerPkg == extraAllowedPkg {
						isCallerPkgAllowed = true
						break
					}
				}
				if isCallerPkgAllowed {
					continue
				}
				pass.Report(analysis.Diagnostic{
					Pos:     n.Pos(),
					Message: fmt.Sprintf(messageFormat, monitoredFunction, pkg, replacement),
				})
			}
		}
	})
	return nil, nil
}
