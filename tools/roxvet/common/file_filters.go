package common

import (
	"go/ast"
	"strconv"
)

// IsTestFile checks if the given file is a test file.
// Note: rather than going off of a _test.go suffix, we check if the file imports the "testing" package.
func IsTestFile(f *ast.File) bool {
	for _, imp := range f.Imports {
		pkgPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			continue
		}
		if pkgPath == "testing" {
			return true
		}
	}
	return false
}
