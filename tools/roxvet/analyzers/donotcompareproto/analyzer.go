package donotcompareproto

import (
	"fmt"
	"go/ast"
	"go/types"
	"regexp"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
	"golang.org/x/tools/go/types/typeutil"
)

const doc = `Inspect calls to Equal for proto arguments that should be compared with protocompat.Equal instead`

// Analyzer is the go vet entrypoint
var Analyzer = &analysis.Analyzer{
	Name:     "donotcompareproto",
	Doc:      doc,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

var (
	protoPkgs = []string{
		"*github.com/stackrox/scanner/generated",
		"*github.com/stackrox/rox/generated/",
	}

	// Support oneof fields. All oneof interfaces have naming pattern "is<FieldName>".
	oneofFieldRegex = regexp.MustCompile(`^github\.com/stackrox/(rox|scanner)/generated/.*\.is[A-Z]`)

	replacements = map[string]string{
		"":              "Equal",
		"[]":            "SlicesEqual",
		"map[string][]": "MapSliceEqual",
		"map[string]":   "MapEqual",
	}

	allowedCallerPackages []string

	bannedEqualFunctions = set.NewFrozenStringSet(
		"github.com/google/go-cmp/cmp.Equal",
		"github.com/google/go-cmp/cmp.Diff",
		"reflect.DeepEqual",
	)

	bannedAssertFunctions = set.NewFrozenStringSet(
		"(*github.com/stretchr/testify/assert.Assertions).Contains",
		"(*github.com/stretchr/testify/assert.Assertions).ElementsMatch",
		"(*github.com/stretchr/testify/assert.Assertions).Equal",
		"(*github.com/stretchr/testify/assert.Assertions).EqualValues",
		"(*github.com/stretchr/testify/assert.Assertions).Equalf",
		"(*github.com/stretchr/testify/assert.Assertions).NotContains",
		"(*github.com/stretchr/testify/assert.Assertions).NotEqual",
		"(*github.com/stretchr/testify/require.Assertions).Contains",
		"(*github.com/stretchr/testify/require.Assertions).ElementsMatch",
		"(*github.com/stretchr/testify/require.Assertions).Equal",
		"(*github.com/stretchr/testify/require.Assertions).EqualValues",
		"(*github.com/stretchr/testify/require.Assertions).Equalf",
		"(*github.com/stretchr/testify/require.Assertions).NotContains",
		"(*github.com/stretchr/testify/require.Assertions).NotEqual",
		"github.com/stretchr/testify/assert.Contains",
		"github.com/stretchr/testify/assert.ElementsMatch",
		"github.com/stretchr/testify/assert.Equal",
		"github.com/stretchr/testify/assert.EqualValues",
		"github.com/stretchr/testify/assert.Equalf",
		"github.com/stretchr/testify/assert.NotContains",
		"github.com/stretchr/testify/assert.NotEqual",
		"github.com/stretchr/testify/require.Contains",
		"github.com/stretchr/testify/require.ElementsMatch",
		"github.com/stretchr/testify/require.Equal",
		"github.com/stretchr/testify/require.EqualValues",
		"github.com/stretchr/testify/require.Equalf",
		"github.com/stretchr/testify/require.NotContains",
		"github.com/stretchr/testify/require.NotEqual",
	)
)

func run(pass *analysis.Pass) (interface{}, error) {
	callerPkg := pass.Pkg.Path()
	for _, allowedPkg := range allowedCallerPackages {
		if strings.HasPrefix(callerPkg, allowedPkg) {
			return nil, nil
		}
	}
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
		name := fn.FullName()
		isBannedAssert := bannedAssertFunctions.Contains(name)
		isBannedEqual := bannedEqualFunctions.Contains(name)
		if !isBannedEqual && !isBannedAssert {
			return
		}

		pkg := "protoutils"
		if isBannedAssert {
			pkg = "protoassert"
		}

		for _, arg := range call.Args[:min(len(call.Args), 3)] {
			typ := pass.TypesInfo.Types[arg].Type
			if typ == nil {
				continue
			}
			// ignore enums
			if typ.Underlying().String() == "int32" {
				continue
			}
			comparedTypeString := typ.String()

			// Ignore Contains that check keys in map
			if strings.Contains(name, "Contains") && strings.HasPrefix(comparedTypeString, "map[string]") {
				continue
			}

			for _, protoPkg := range protoPkgs {
				for modifier, r := range replacements {
					if strings.HasPrefix(comparedTypeString, modifier+protoPkg) {
						pass.Report(analysis.Diagnostic{
							Pos:     arg.Pos(),
							Message: fmt.Sprintf("Do not use %s on proto.Message, use %s.%s", name, pkg, r),
						})
						return
					}
				}

				if strings.Contains(comparedTypeString, protoPkg) {
					pass.Report(analysis.Diagnostic{
						Pos:     arg.Pos(),
						Message: fmt.Sprintf("Do not use %s on proto.Message", name),
					})
					return
				}
			}

			if oneofFieldRegex.MatchString(comparedTypeString) {
				pass.Report(analysis.Diagnostic{
					Pos:     arg.Pos(),
					Message: fmt.Sprintf("Do not use %s on proto 'oneof' fields, use provided functions in %s package and compare relevant field(s) from 'oneof' list", name, pkg),
				})
				return
			}
		}
	})
	return nil, nil
}
