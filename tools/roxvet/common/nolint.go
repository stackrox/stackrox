package common

import (
	"strings"

	"github.com/stackrox/rox/pkg/set"
	"golang.org/x/tools/go/analysis"
)

// NolintPositions returns all positions to end of statements that should not be linted by linter with linterName.
func NolintPositions(pass *analysis.Pass, linterName string) set.IntSet {
	nolint := set.IntSet{}
	for _, f := range pass.Files {
		for _, comment := range f.Comments {
			for _, c := range comment.List {
				if strings.HasPrefix(c.Text, "//nolint:"+linterName) {
					nolint.Add(int(c.Pos()) - 1)
				}
			}
		}
	}
	return nolint
}
