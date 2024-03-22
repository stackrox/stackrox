package common

import (
	"go/ast"
	"regexp"
	"strconv"
)

var (
	generatedCodeRegex = regexp.MustCompile(`// Code generated .* DO NOT EDIT\.$`)
)

// IsGeneratedFile checks if the given file is a generated file, as indicated by the generated file header.
func IsGeneratedFile(f *ast.File) bool {
	for _, comments := range f.Comments {
		for _, comment := range comments.List {
			// This line must appear before the first non-comment, non-blank text in the file.
			if comment.Pos() > f.Package {
				return false
			}
			if generatedCodeRegex.MatchString(comment.Text) {
				return true
			}
		}
	}
	return false
}

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
