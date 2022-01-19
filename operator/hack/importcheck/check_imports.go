// This program checks that go files listed as arguments import certain packages
// using a particular name for consistency.
package main

import (
	"fmt"
	parser "go/parser"
	"go/token"
	"os"
	"strings"
)

var rules = []struct {
	description      string
	importPathSuffix string
	expectedName     string
}{
	{
		"controller-runtime client",
		"controller-runtime/pkg/client\"",
		"ctrlClient",
	},
}

func main() {
	failure := false
	for _, filename := range os.Args[1:] {
		if !checkFile(filename) {
			failure = true
		}
	}
	if failure {
		os.Exit(1)
	}
}

func checkFile(filename string) bool {
	fileSet := token.NewFileSet()
	f, err := parser.ParseFile(fileSet, filename, nil, 0)
	if err != nil {
		panic(err)
	}
	for _, importSpec := range f.Imports {
		for _, rule := range rules {
			if !strings.HasSuffix(importSpec.Path.Value, rule.importPathSuffix) {
				continue
			}
			if importSpec.Name == nil || importSpec.Name.Name != rule.expectedName {
				fmt.Printf("Please import %s as %q in %s\n", rule.description, rule.expectedName, fileSet.Position(importSpec.Pos()))
				return false
			}
		}
	}
	return true
}
