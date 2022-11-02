package common

import (
	"go/ast"
)

// FileFilter defines a filter for _positively_ selecting files during AST traversal (i.e., a file is included
// in the traversal if the filter function returns true).
type FileFilter func(*ast.File) bool

// Not inverts a given file filter
func Not(ff FileFilter) FileFilter {
	return func(f *ast.File) bool {
		return !ff(f)
	}
}

// And returns the logical conjunction of the given filters.
func And(ffs ...FileFilter) FileFilter {
	return func(f *ast.File) bool {
		for _, ff := range ffs {
			if !ff(f) {
				return false
			}
		}
		return true
	}
}

// Or returns the logical disjunction of the given filters.
func Or(ffs ...FileFilter) FileFilter {
	return func(f *ast.File) bool {
		for _, ff := range ffs {
			if ff(f) {
				return true
			}
		}
		return false
	}
}
