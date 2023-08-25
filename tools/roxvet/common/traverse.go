package common

import (
	"go/ast"

	"golang.org/x/tools/go/ast/inspector"
)

// FilteredPreorder calls `inspector.Preorder(nodeTypes, fn)`, but filters out all files that do not pass the given
// fileFilter.
func FilteredPreorder(inspector *inspector.Inspector, fileFilter FileFilter, nodeTypes []ast.Node, fn func(n ast.Node)) {
	// Limit the nodeTypes slice's capacity to its length. This ensures that any call to `append` will not write into
	// the backing storage, but allocate a new slice.
	effectiveTypes := nodeTypes[:len(nodeTypes):len(nodeTypes)]
	hadFile := hasFile(nodeTypes)
	if !hadFile {
		effectiveTypes = append(effectiveTypes, (*ast.File)(nil))
	}
	var skip bool
	inspector.Preorder(effectiveTypes, func(n ast.Node) {
		if f, ok := n.(*ast.File); ok {
			skip = !fileFilter(f)
			if !hadFile {
				return
			}
		}
		if !skip {
			fn(n)
		}
	})
}

// FilteredNodes calls `inspector.Nodes(nodeTypes, fn)`, but filters out all files that do not pass the given
// fileFilter.
func FilteredNodes(inspector *inspector.Inspector, fileFilter FileFilter, nodeTypes []ast.Node, fn func(n ast.Node, push bool) bool) {
	effTypes := nodeTypes[:len(nodeTypes):len(nodeTypes)]
	hadFile := hasFile(nodeTypes)
	if !hadFile {
		effTypes = append(effTypes, (*ast.File)(nil))
	}
	inspector.Nodes(effTypes, func(n ast.Node, push bool) bool {
		if f, ok := n.(*ast.File); ok {
			if !push {
				if hadFile {
					return fn(n, push)
				}
				return false
			}

			if !fileFilter(f) {
				return false
			}
			if !hadFile {
				return true
			}
		}
		return fn(n, push)
	})
}

func hasFile(types []ast.Node) bool {
	if types == nil {
		return true // nil indicates all node types
	}
	for _, t := range types {
		if _, ok := t.(*ast.File); ok {
			return true
		}
	}
	return false
}
