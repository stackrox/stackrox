package policyreport

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllPolicyReportResourcesAreAddedToTheAvailabilityChecker(t *testing.T) {
	ac := NewAvailabilityChecker()
	pwd, err := os.Getwd()
	require.NoError(t, err)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path.Join(pwd, "../../../../../pkg/policyreport/api.go"), nil, 0)
	require.NoError(t, err)

	allowList := []string{
		"groupVersion",
		"requiredAPIResources",
	}

	resFinder := &resourcesFinder{}
	ast.Walk(resFinder, file)
	require.NotEmpty(t, resFinder.resources)

	var notFound []string
finderLoop:
	for _, resource := range resFinder.resources {
		for _, acResource := range ac.GetResources() {
			if acResource.Kind == resource {
				continue finderLoop
			}
		}
		for _, allowed := range allowList {
			if allowed == resource {
				continue finderLoop
			}
		}
		notFound = append(notFound, resource)
	}

	assert.Empty(t, notFound, "Please add the missing types to the availability checker or to the allowList in this test")
}

type resourcesFinder struct {
	resources []string
}

func (f *resourcesFinder) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.Package:
		return f
	case *ast.File:
		return f
	case *ast.GenDecl:
		if n.Tok == token.VAR {
			return f
		}
	case *ast.ValueSpec:
		if len(n.Names) < 1 {
			return nil
		}
		f.resources = append(f.resources, n.Names[0].Name)
	}
	return nil
}
