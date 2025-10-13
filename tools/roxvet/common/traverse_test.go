package common

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stackrox/rox/pkg/sliceutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	file1 = `
package file1pkg

// This is file1

func funcInFile1() int {
	return 42
}
`
	file2 = `
package file2pkg

// This is file2

func funcInFile2() int {
	return 42
}
`
)

func TestFilteredTraversal(t *testing.T) {
	fset := token.NewFileSet()
	file1, err := parser.ParseFile(fset, "file1.go", file1, parser.ParseComments)
	require.NoError(t, err)

	file2, err := parser.ParseFile(fset, "file2.go", file2, parser.ParseComments)
	require.NoError(t, err)

	files := []*ast.File{file1, file2}

	filterFuncs := map[string]FileFilter{
		"all files":  func(f *ast.File) bool { return true },
		"file1 only": func(f *ast.File) bool { return f.Name.Name == "file1pkg" },
		"file2 only": func(f *ast.File) bool { return f.Name.Name == "file2pkg" },
		"no files":   func(f *ast.File) bool { return false },
	}

	for name, ff := range filterFuncs {
		t.Run(fmt.Sprintf("with filter %q", name), func(t *testing.T) {
			filteredFiles := sliceutils.Filter(files, ff)

			// The following tests all test the equivalence between running Filtered<Traversal> on all files and
			// running <Traversal> on the filtered set of files, using the current file filter and either no filter
			// on ast.Node types, or a filter including identifiers only.

			filteredFilesInspector := inspector.New(filteredFiles)
			allFilesInspector := inspector.New(files)

			t.Run("preorder, all nodes", func(t *testing.T) {
				var nodesInFilteredFiles []ast.Node
				filteredFilesInspector.Preorder(nil, func(n ast.Node) {
					nodesInFilteredFiles = append(nodesInFilteredFiles, n)
				})

				var nodesInFilteredTraversal []ast.Node
				FilteredPreorder(allFilesInspector, ff, nil, func(n ast.Node) {
					nodesInFilteredTraversal = append(nodesInFilteredTraversal, n)
				})

				assert.Equal(t, nodesInFilteredFiles, nodesInFilteredTraversal)
			})

			t.Run("preorder, identifiers only", func(t *testing.T) {
				nodeFilter := []ast.Node{(*ast.Ident)(nil)}
				var nodesInFilteredFiles []ast.Node
				filteredFilesInspector.Preorder(nodeFilter, func(n ast.Node) {
					nodesInFilteredFiles = append(nodesInFilteredFiles, n)
				})

				var nodesInFilteredTraversal []ast.Node
				FilteredPreorder(allFilesInspector, ff, nodeFilter, func(n ast.Node) {
					nodesInFilteredTraversal = append(nodesInFilteredTraversal, n)
				})

				assert.Equal(t, nodesInFilteredFiles, nodesInFilteredTraversal)
			})

			type nodeWithPush struct {
				node ast.Node
				push bool
			}

			t.Run("nodes, all nodes", func(t *testing.T) {
				var nodesInFilteredFiles []nodeWithPush
				filteredFilesInspector.Nodes(nil, func(n ast.Node, push bool) bool {
					nodesInFilteredFiles = append(nodesInFilteredFiles, nodeWithPush{node: n, push: push})
					return true
				})

				var nodesInFilteredTraversal []nodeWithPush
				FilteredNodes(allFilesInspector, ff, nil, func(n ast.Node, push bool) bool {
					nodesInFilteredTraversal = append(nodesInFilteredTraversal, nodeWithPush{node: n, push: push})
					return true
				})

				assert.Equal(t, nodesInFilteredFiles, nodesInFilteredTraversal)
			})

			t.Run("nodes, identifiers only", func(t *testing.T) {
				nodeFilter := []ast.Node{(*ast.Ident)(nil)}

				var nodesInFilteredFiles []nodeWithPush
				filteredFilesInspector.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
					nodesInFilteredFiles = append(nodesInFilteredFiles, nodeWithPush{node: n, push: push})
					return true
				})

				var nodesInFilteredTraversal []nodeWithPush
				FilteredNodes(allFilesInspector, ff, nodeFilter, func(n ast.Node, push bool) bool {
					nodesInFilteredTraversal = append(nodesInFilteredTraversal, nodeWithPush{node: n, push: push})
					return true
				})

				assert.Equal(t, nodesInFilteredFiles, nodesInFilteredTraversal)
			})
		})
	}
}
