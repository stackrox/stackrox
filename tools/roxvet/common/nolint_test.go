package common

import (
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestNoLint(t *testing.T) {

	files := map[string]string{
		"p/a.go": `package p //nolint:test`,
		"p/b.go": `package p //nolint:other`,
		"p/c.go": `package p //nolint:test`,
	}
	const name = "test"

	test := &analysis.Analyzer{
		Name: name,
		Doc:  "dummy analyzer just to get Pass object",
		Run: func(p *analysis.Pass) (interface{}, error) {
			return nil, nil
		},
	}

	testdata, cleanup, err := analysistest.WriteFiles(files)
	assert.NoError(t, err)
	t.Cleanup(cleanup)
	path := filepath.Join(testdata, "./...")
	results := analysistest.RunWithSuggestedFixes(t, testdata, test, path)
	assert.Len(t, results, 1)
	actual := NolintPositions(results[0].Pass, name)
	// Since files may be analysed in any order we need to check all possible scenarios.
	// As a and b are the same we don't need to check ab and ba order.
	assert.True(t,
		actual.Equal(set.NewIntSet(10, 34)) || // a, c, b
			actual.Equal(set.NewIntSet(10, 59)) || // a, b, c
			actual.Equal(set.NewIntSet(35, 59)), // b, a, c
		actual.AsSlice())
}
