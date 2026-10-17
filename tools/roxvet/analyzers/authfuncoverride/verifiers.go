package authfuncoverride

import (
	"go/ast"
	"go/types"
	"reflect"

	"github.com/stackrox/rox/pkg/grpc/authz"
	"github.com/stackrox/rox/pkg/grpc/authz/allow"
	authzand "github.com/stackrox/rox/pkg/grpc/authz/and"
	"github.com/stackrox/rox/pkg/grpc/authz/deny"
	"github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	authzor "github.com/stackrox/rox/pkg/grpc/authz/or"
	"github.com/stackrox/rox/pkg/grpc/authz/user"
	"golang.org/x/tools/go/analysis"
)

const (
	userPkgPath    = "github.com/stackrox/rox/pkg/grpc/authz/user"
	allowPkgPath   = "github.com/stackrox/rox/pkg/grpc/authz/allow"
	denyPkgPath    = "github.com/stackrox/rox/pkg/grpc/authz/deny"
	andPkgPath     = "github.com/stackrox/rox/pkg/grpc/authz/and"
	idcheckPkgPath = "github.com/stackrox/rox/pkg/grpc/authz/idcheck"
	orPkgPath      = "github.com/stackrox/rox/pkg/grpc/authz/or"
	perrpcPkgPath  = "github.com/stackrox/rox/pkg/grpc/authz/perrpc"
)

// allowedConstructor identifies a leaf authorizer constructor by package and function name.
// perrpc.FromMap and the or.SensorOr/or.ScannerOr wrappers are handled separately because
// they require recursive validation of their arguments.
type allowedConstructor struct {
	pkgPath  string
	funcName string
}

// leafConstructors is the complete set of accepted leaf constructors.
// Constructors that require recursive argument validation are NOT listed here.
var leafConstructors = []allowedConstructor{
	// user-space authorizers
	{pkgPath: userPkgPath, funcName: "WithRole"},
	{pkgPath: userPkgPath, funcName: "With"},
	{pkgPath: userPkgPath, funcName: "Authenticated"},
	// allow
	{pkgPath: allowPkgPath, funcName: "Anonymous"},
	// deny
	{pkgPath: denyPkgPath, funcName: "Everyone"},
	// or-based composite helpers accepted as leaves (their arguments are hard-coded safe)
	{pkgPath: orPkgPath, funcName: "ScannerV4"},
	// identity-based service checks
	{pkgPath: idcheckPkgPath, funcName: "CentralOnly"},
	{pkgPath: idcheckPkgPath, funcName: "SensorsOnly"},
	{pkgPath: idcheckPkgPath, funcName: "SensorRegistrantsOnly"},
	{pkgPath: idcheckPkgPath, funcName: "ScannerOnly"},
	{pkgPath: idcheckPkgPath, funcName: "ScannerV4IndexerOnly"},
	{pkgPath: idcheckPkgPath, funcName: "ScannerV4MatcherOnly"},
	{pkgPath: idcheckPkgPath, funcName: "CollectorOnly"},
	{pkgPath: idcheckPkgPath, funcName: "AdmissionControlOnly"},
}

// orCompositeConstructors are or-package wrappers that accept a single authz.Authorizer
// argument which must itself be a valid authorizer.
var orCompositeConstructors = map[string]bool{
	"SensorOr":  true,
	"ScannerOr": true,
}

// allowedConcreteTypes holds the reflect.Type of the concrete value returned by each
// distinct family of allowed constructors, captured once at init time.
//
// Notes:
//   - user.Authenticated() delegates to user.With() and returns the same *permChecker
//     type, so no separate entry is needed.
//   - All idcheck.XXXOnly() functions return the same identityBasedAuthorizerWrapper
//     type; CentralOnly() is used as the representative.
var allowedConcreteTypes = []reflect.Type{
	reflect.TypeOf(user.WithRole("")),     // *user.roleChecker
	reflect.TypeOf(user.With()),           // *user.permChecker — also covers Authenticated()
	reflect.TypeOf(allow.Anonymous()),     // allow.anonymous
	reflect.TypeOf(deny.Everyone()),       // deny.everyone
	reflect.TypeOf(idcheck.CentralOnly()), // idcheck.identityBasedAuthorizerWrapper — covers all XXXOnly()
	reflect.TypeOf(authzor.ScannerV4()),   // or.orAuthorizer — also covers Or, SensorOr, ScannerOr
	reflect.TypeOf(authzand.And()),        // and.andAuthorizer
}

// IsWithRole reports whether a is the concrete type returned by user.WithRole.
// Intended for use in tests and runtime assertions, not in the static linter.
func IsWithRole(a authz.Authorizer) bool {
	return reflect.TypeOf(a) == allowedConcreteTypes[0]
}

// IsAllowedAuthorizer reports whether a was constructed by one of the allowed
// constructors: user.WithRole, user.With / user.Authenticated, allow.Anonymous,
// deny.Everyone, any idcheck.XXXOnly function, or.ScannerV4 / or.Or / or.SensorOr /
// or.ScannerOr, or and.And.
// Note: the runtime check cannot validate arguments passed to variadic / composite
// constructors (Or, And, SensorOr, ScannerOr) — use the static linter for that.
// Intended for use in tests and runtime assertions, not in the static linter.
func IsAllowedAuthorizer(a authz.Authorizer) bool {
	t := reflect.TypeOf(a)
	for _, allowed := range allowedConcreteTypes {
		if t == allowed {
			return true
		}
	}
	return false
}

// UserAuthorizerVerifier is an AuthorizerVerifier that accepts authorizers constructed
// by one of the following constructors:
//
//   - user.WithRole / user.With / user.Authenticated (pkg/grpc/authz/user)
//   - allow.Anonymous (pkg/grpc/authz/allow)
//   - deny.Everyone (pkg/grpc/authz/deny)
//   - idcheck.CentralOnly / SensorsOnly / SensorRegistrantsOnly / ScannerOnly /
//     ScannerV4IndexerOnly / ScannerV4MatcherOnly / CollectorOnly / AdmissionControlOnly
//     (pkg/grpc/authz/idcheck)
//   - or.ScannerV4 (pkg/grpc/authz/or)
//   - or.Or / or.SensorOr / or.ScannerOr (pkg/grpc/authz/or), provided that all
//     authorizer arguments are themselves valid (recursively checked).
//   - and.And (pkg/grpc/authz/and), provided that all authorizer arguments are
//     themselves valid (recursively checked).
//   - perrpc.FromMap (pkg/grpc/authz/perrpc), provided that every key in the map
//     literal is itself a valid authorizer (recursively checked).
//
// Named variables are resolved to their initializer within the current package.
// Inline call expressions are checked directly. Cross-package variables are not
// supported (the verifier conservatively rejects them).
func UserAuthorizerVerifier(authorizerExpr ast.Expr, pass *analysis.Pass) bool {
	return isValidAuthorizerExpr(authorizerExpr, pass)
}

// isValidAuthorizerExpr is the recursive core of the verifier.
func isValidAuthorizerExpr(expr ast.Expr, pass *analysis.Pass) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return isAllowedConstructorCall(e, pass)
	case *ast.Ident:
		obj := pass.TypesInfo.Uses[e]
		if obj == nil {
			return false
		}
		init := findVarInitializer(obj, pass.Files)
		if init == nil {
			return false
		}
		return isValidAuthorizerExpr(init, pass)
	}
	return false
}

// isAllowedConstructorCall returns true when call is one of the accepted constructors.
// For perrpc.FromMap it recursively validates all map-literal keys.
// For or.Or / and.And it recursively validates all variadic arguments.
// For or.SensorOr / or.ScannerOr it recursively validates their single argument.
func isAllowedConstructorCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkgIdent, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName, ok := pass.TypesInfo.Uses[pkgIdent].(*types.PkgName)
	if !ok {
		return false
	}
	importedPath := pkgName.Imported().Path()
	funcName := sel.Sel.Name

	if importedPath == perrpcPkgPath && funcName == "FromMap" {
		return isValidFromMapCall(call, pass)
	}
	if (importedPath == orPkgPath && funcName == "Or") ||
		(importedPath == andPkgPath && funcName == "And") {
		return isValidVariadicCall(call, pass)
	}
	if importedPath == orPkgPath && orCompositeConstructors[funcName] {
		return isValidSingleArgOrCall(call, pass)
	}
	for _, c := range leafConstructors {
		if importedPath == c.pkgPath && funcName == c.funcName {
			return true
		}
	}
	return false
}

// isValidFromMapCall checks that a perrpc.FromMap(map[authz.Authorizer][]string{...})
// call uses only valid authorizers as map keys.
func isValidFromMapCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	if len(call.Args) != 1 {
		return false
	}
	mapLit, ok := call.Args[0].(*ast.CompositeLit)
	if !ok {
		return false
	}
	for _, elt := range mapLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return false
		}
		if !isValidAuthorizerExpr(kv.Key, pass) {
			return false
		}
	}
	return true
}

// isValidVariadicCall checks that every argument in a variadic constructor call
// (or.Or, and.And) is itself a valid authorizer.
func isValidVariadicCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	for _, arg := range call.Args {
		if !isValidAuthorizerExpr(arg, pass) {
			return false
		}
	}
	return true
}

// isValidSingleArgOrCall checks that an or.SensorOr / or.ScannerOr call is given
// a single authorizer argument that is itself a valid authorizer.
func isValidSingleArgOrCall(call *ast.CallExpr, pass *analysis.Pass) bool {
	if len(call.Args) != 1 {
		return false
	}
	return isValidAuthorizerExpr(call.Args[0], pass)
}

// findVarInitializer searches the provided AST files for the var declaration that
// defines obj and returns its initializer expression, or nil if not found.
func findVarInitializer(obj types.Object, files []*ast.File) ast.Expr {
	objPos := obj.Pos()
	for _, file := range files {
		var result ast.Expr
		ast.Inspect(file, func(n ast.Node) bool {
			if result != nil {
				return false
			}
			spec, ok := n.(*ast.ValueSpec)
			if !ok {
				return true
			}
			for i, name := range spec.Names {
				if name.Pos() == objPos && i < len(spec.Values) {
					result = spec.Values[i]
					return false
				}
			}
			return true
		})
		if result != nil {
			return result
		}
	}
	return nil
}
