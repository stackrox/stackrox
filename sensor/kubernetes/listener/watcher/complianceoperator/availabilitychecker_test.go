package complianceoperator

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

func TestAllCOResourcesAreAddedToTheAvailabilityChecker(t *testing.T) {
	ac := NewComplianceOperatorAvailabilityChecker()
	pwd, err := os.Getwd()
	require.NoError(t, err)

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path.Join(pwd, "../../../../../pkg/complianceoperator/api.go"), nil, 0)
	require.NoError(t, err)

	whileList := []string{
		"groupVersion",
		"requiredAPIResources",
	}

	resFinder := &resourcesFinder{}
	ast.Walk(resFinder, file)
	require.NotEmpty(t, resFinder.resources)

	var notFound []string
finderLoop:
	for _, resource := range resFinder.resources {
		for _, acResource := range ac.resources {
			if acResource.Kind == resource {
				continue finderLoop
			}
		}
		for _, whileListed := range whileList {
			if whileListed == resource {
				continue finderLoop
			}
		}
		notFound = append(notFound, resource)
	}

	assert.Empty(t, notFound, "Please add the missing types to the resources field in the availability checker to the whilelist in this test if they should not be used in the availability checker")
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
		// This should never happen
		if len(n.Names) < 1 {
			return nil
		}
		f.resources = append(f.resources, n.Names[0].Name)
	}
	return nil
}
