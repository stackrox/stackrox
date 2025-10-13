package storeinterface

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
)

const (
	roxPrefix             = "github.com/stackrox/rox"
	generatedImportPrefix = roxPrefix + "/generated"
	storageImportPrefix   = generatedImportPrefix + "/storage"
)

var (
	// These are proto types returned from Stores which are not stored there, and so don't have to be in storage.
	allowedList = set.NewFrozenStringSet(
		"v1.SearchResult",
		"v1.HostResults",
		"v1.ImportPolicyResponse",
	)
)

// Analyzer is the analyzer exported by this package.
var Analyzer = &analysis.Analyzer{
	Name: "storeinterface",
	Doc: `Looks for interfaces whose names end in Store, which return proto objects not in storage.
Specifically, if any of the function definitions in the interface return an object that's defined in proto-generated-code NOT
in the storage folder, this analyzer will report it.'`,
	Run: run,
}

// Return all imports from the generated path except github.com/stackrox/rox/generated/storage
// It returns whatever the import will be referenced as locally.
func localPackageNamesForGeneratedImports(typeInfo *types.Info, f *ast.File) set.StringSet {
	packageNames := set.NewStringSet()
	for _, spec := range f.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			panic(err)
		}
		if !strings.HasPrefix(path, generatedImportPrefix) {
			continue
		}
		if path == storageImportPrefix {
			continue
		}
		if spec.Name != nil {
			packageNames.Add(spec.Name.String())
			continue
		}
		pkg := imported(typeInfo, spec)
		packageNames.Add(pkg.Name())
	}
	return packageNames
}

// retrieves all definitions of interface (type <blah> interface) where the name of the interface
// ends in Store.
func retrieveStoreInterfaces(f *ast.File) (interfaces map[string]*ast.InterfaceType) {
	interfaces = make(map[string]*ast.InterfaceType)
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		if genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec := spec.(*ast.TypeSpec)
			typeSpecName := typeSpec.Name.String()
			if !strings.HasSuffix(typeSpecName, "Store") {
				continue
			}
			if interfaceType, ok := typeSpec.Type.(*ast.InterfaceType); ok {
				interfaces[typeSpecName] = interfaceType
			}
		}
	}
	return
}

// Takes an expression and returns the qualifier and identifier, if it is an expression that contains one.
// Example: returns v1.Deployment if the expr is either v1.Deployment, or *v1.Deployment, or []*v1.Deployment
func qualifierAndIdentifierFromExpr(expr ast.Expr) (qualifier, identifier string) {
	if arrayExpr, ok := expr.(*ast.ArrayType); ok {
		return qualifierAndIdentifierFromExpr(arrayExpr.Elt)
	}
	if starExpr, ok := expr.(*ast.StarExpr); ok {
		return qualifierAndIdentifierFromExpr(starExpr.X)
	}
	if asSelector, ok := expr.(*ast.SelectorExpr); ok {
		return asSelector.X.(*ast.Ident).Name, asSelector.Sel.Name
	}
	return "", ""
}

// If the interface is returning any values that are imported from the set of passed packageNames, this function returns it.
// Example, if we have a store:
//
//	type Store interface {
//	   A() (*v1.Deployment)
//
// and packageNames contains `v1`,
// we would return v1.Deployment since it is from the list of forbidden package names.
func returnValuesFromForbiddenPackage(forbiddenPackageNames set.StringSet, interfaceType *ast.InterfaceType) string {
	for _, method := range interfaceType.Methods.List {
		ft, ok := method.Type.(*ast.FuncType)
		if !ok {
			// This happens if there's an embedded type in the interface.
			continue
		}
		if ft.Results == nil {
			continue
		}
		for _, result := range ft.Results.List {
			if qualifier, identifier := qualifierAndIdentifierFromExpr(result.Type); qualifier != "" {
				if forbiddenPackageNames.Contains(qualifier) {
					qualifiedExpression := fmt.Sprintf("%s.%s", qualifier, identifier)
					if !allowedList.Contains(qualifiedExpression) {
						return fmt.Sprintf("%s.%s", qualifier, identifier)
					}
				}
			}
		}
	}
	return ""
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, f := range pass.Files {
		imports := localPackageNamesForGeneratedImports(pass.TypesInfo, f)
		for name, interfaceType := range retrieveStoreInterfaces(f) {
			if matchingIdentifier := returnValuesFromForbiddenPackage(imports, interfaceType); matchingIdentifier != "" {
				pass.Reportf(interfaceType.Pos(), "interface %s in package %s seems to store value %s, which "+
					"is a proto not in storage",
					name, pass.Pkg, matchingIdentifier)
			}
		}
	}
	return nil, nil
}

func imported(typeInfo *types.Info, spec *ast.ImportSpec) *types.Package {
	obj, ok := typeInfo.Implicits[spec]
	if !ok {
		obj = typeInfo.Defs[spec.Name] // renaming import
	}
	return obj.(*types.PkgName).Imported()
}
