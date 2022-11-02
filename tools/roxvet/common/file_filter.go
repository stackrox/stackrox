package common

import "go/ast"

// FileFilter defines a filter for _positively_ selecting files during AST traversal (i.e., a file is included
// in the traversal if the filter function returns true).
type FileFilter func(*ast.File) bool

// Not inverts a given file filter
func Not(ff FileFilter) FileFilter {
	return func(f *ast.File) bool {
		return !ff(f)
	}
}
