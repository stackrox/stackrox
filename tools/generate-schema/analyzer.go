package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// TypeInfo represents analyzed information about a Go type
type TypeInfo struct {
	Name        string
	PackagePath string
	Fields      []FieldInfo
}

// FieldInfo represents analyzed information about a struct field
type FieldInfo struct {
	Name         string
	Type         string
	Kind         reflect.Kind
	Tag          string
	JsonTag      string
	SqlTag       string
	SearchTag    string
	ProtoTag     string
	IsPointer    bool
	IsSlice      bool
	ElementType  string
	ElementKind  reflect.Kind
}

// TypeAnalyzer analyzes Go types using AST and type information
type TypeAnalyzer struct {
	packages map[string]*packages.Package
	fset     *token.FileSet
}

// NewTypeAnalyzer creates a new type analyzer
func NewTypeAnalyzer() *TypeAnalyzer {
	return &TypeAnalyzer{
		packages: make(map[string]*packages.Package),
		fset:     token.NewFileSet(),
	}
}

// LoadPackage loads a Go package for analysis
func (ta *TypeAnalyzer) LoadPackage(packagePath string) error {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax,
		Fset: ta.fset,
		Dir:  ".", // Use current directory as working directory
	}

	pkgs, err := packages.Load(cfg, packagePath)
	if err != nil {
		return fmt.Errorf("loading package %s: %w", packagePath, err)
	}

	if len(pkgs) == 0 {
		return fmt.Errorf("no packages found for %s", packagePath)
	}

	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return fmt.Errorf("package %s has errors: %v", packagePath, pkg.Errors)
	}

	ta.packages[packagePath] = pkg
	return nil
}

// AnalyzeType analyzes a specific type within a loaded package
func (ta *TypeAnalyzer) AnalyzeType(packagePath, typeName string) (*TypeInfo, error) {
	pkg, exists := ta.packages[packagePath]
	if !exists {
		return nil, fmt.Errorf("package %s not loaded", packagePath)
	}

	// Find the type in the package
	obj := pkg.Types.Scope().Lookup(typeName)
	if obj == nil {
		return nil, fmt.Errorf("type %s not found in package %s", typeName, packagePath)
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		return nil, fmt.Errorf("type %s is not a named type", typeName)
	}

	underlying, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil, fmt.Errorf("type %s is not a struct", typeName)
	}

	// Find the AST node for this type to get struct tags
	var astStruct *ast.StructType
	for _, file := range pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
				if structType, ok := ts.Type.(*ast.StructType); ok {
					astStruct = structType
					return false
				}
			}
			return true
		})
		if astStruct != nil {
			break
		}
	}

	if astStruct == nil {
		return nil, fmt.Errorf("could not find AST for struct %s", typeName)
	}

	typeInfo := &TypeInfo{
		Name:        typeName,
		PackagePath: packagePath,
		Fields:      make([]FieldInfo, 0, underlying.NumFields()),
	}

	// Analyze struct fields
	for i := 0; i < underlying.NumFields(); i++ {
		field := underlying.Field(i)
		var tag string

		// Get the tag from the AST
		if i < len(astStruct.Fields.List) {
			astField := astStruct.Fields.List[i]
			if astField.Tag != nil {
				tag = strings.Trim(astField.Tag.Value, "`")
			}
		}

		fieldInfo := ta.analyzeField(field, tag)
		typeInfo.Fields = append(typeInfo.Fields, fieldInfo)
	}

	return typeInfo, nil
}

// analyzeField analyzes a single struct field
func (ta *TypeAnalyzer) analyzeField(field *types.Var, tag string) FieldInfo {
	fieldInfo := FieldInfo{
		Name: field.Name(),
		Tag:  tag,
	}

	// Parse struct tag
	if tag != "" {
		fieldInfo.JsonTag = ta.parseTag(tag, "json")
		fieldInfo.SqlTag = ta.parseTag(tag, "sql")
		fieldInfo.SearchTag = ta.parseTag(tag, "search")
		fieldInfo.ProtoTag = ta.parseTag(tag, "protobuf")
	}

	// Analyze field type
	ta.analyzeFieldType(field.Type(), &fieldInfo)

	return fieldInfo
}

// analyzeFieldType recursively analyzes the type of a field
func (ta *TypeAnalyzer) analyzeFieldType(t types.Type, fieldInfo *FieldInfo) {
	switch typ := t.(type) {
	case *types.Pointer:
		fieldInfo.IsPointer = true
		ta.analyzeFieldType(typ.Elem(), fieldInfo)
	case *types.Slice:
		fieldInfo.IsSlice = true
		// For slices, analyze the element type
		ta.analyzeElementType(typ.Elem(), fieldInfo)
		// Set the main type as slice
		fieldInfo.Type = fmt.Sprintf("[]%s", fieldInfo.ElementType)
		fieldInfo.Kind = reflect.Slice
	case *types.Basic:
		fieldInfo.Type = typ.String()
		fieldInfo.Kind = ta.basicTypeToKind(typ)
	case *types.Named:
		fieldInfo.Type = typ.String()
		// Check if it's an enum (int32 with methods) or struct
		if underlying, ok := typ.Underlying().(*types.Basic); ok {
			fieldInfo.Kind = ta.basicTypeToKind(underlying)
		} else if _, ok := typ.Underlying().(*types.Struct); ok {
			fieldInfo.Kind = reflect.Struct
		} else {
			fieldInfo.Kind = reflect.Interface
		}
	case *types.Interface:
		fieldInfo.Type = typ.String()
		fieldInfo.Kind = reflect.Interface
	default:
		fieldInfo.Type = t.String()
		fieldInfo.Kind = reflect.Interface
	}
}

// analyzeElementType analyzes the element type for slices
func (ta *TypeAnalyzer) analyzeElementType(t types.Type, fieldInfo *FieldInfo) {
	switch typ := t.(type) {
	case *types.Pointer:
		// For pointer element types, get the underlying type
		ta.analyzeElementType(typ.Elem(), fieldInfo)
	case *types.Basic:
		fieldInfo.ElementType = typ.String()
		fieldInfo.ElementKind = ta.basicTypeToKind(typ)
	case *types.Named:
		fieldInfo.ElementType = typ.String()
		if underlying, ok := typ.Underlying().(*types.Basic); ok {
			fieldInfo.ElementKind = ta.basicTypeToKind(underlying)
		} else if _, ok := typ.Underlying().(*types.Struct); ok {
			fieldInfo.ElementKind = reflect.Struct
		} else {
			fieldInfo.ElementKind = reflect.Interface
		}
	default:
		fieldInfo.ElementType = t.String()
		fieldInfo.ElementKind = reflect.Interface
	}
}

// basicTypeToKind converts a types.Basic to reflect.Kind
func (ta *TypeAnalyzer) basicTypeToKind(basic *types.Basic) reflect.Kind {
	switch basic.Kind() {
	case types.Bool:
		return reflect.Bool
	case types.String:
		return reflect.String
	case types.Int8:
		return reflect.Int8
	case types.Int16:
		return reflect.Int16
	case types.Int32:
		return reflect.Int32
	case types.Int64:
		return reflect.Int64
	case types.Int:
		return reflect.Int
	case types.Uint8:
		return reflect.Uint8
	case types.Uint16:
		return reflect.Uint16
	case types.Uint32:
		return reflect.Uint32
	case types.Uint64:
		return reflect.Uint64
	case types.Uint:
		return reflect.Uint
	case types.Float32:
		return reflect.Float32
	case types.Float64:
		return reflect.Float64
	default:
		return reflect.Interface
	}
}

// parseTag parses a specific tag from a struct tag string
func (ta *TypeAnalyzer) parseTag(tag, key string) string {
	// Simple tag parsing - this could be enhanced to use reflect.StructTag
	parts := strings.Split(tag, " ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, key+":") {
			value := strings.TrimPrefix(part, key+":")
			value = strings.Trim(value, "\"")
			return value
		}
	}
	return ""
}