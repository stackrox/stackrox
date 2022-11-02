package common

import (
	"go/ast"
	"regexp"
)

var (
	generatedCodeRegex = regexp.MustCompile(`// Code generated .* DO NOT EDIT\.$`)
)

// IsGeneratedFile checks if the given file is a generated file, as indicated by the generated file header.
func IsGeneratedFile(f *ast.File) bool {
	if len(f.Comments) == 0 {
		return false
	}
	firstComment := f.Comments[0].List[0]

	if firstComment.Pos() > f.Package {
		// comment is not at beginning of file, before the package keyword
		return false
	}

	return generatedCodeRegex.MatchString(firstComment.Text)
}
