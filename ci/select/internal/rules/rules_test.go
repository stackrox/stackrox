package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	data := []byte(`
version: 1
jobs: [go, postgres]
always_run: [lint]
rules:
  - name: go-unit
    jobs: [go]
    skip_if_only_changed: ['^ui/']
  - name: db
    jobs: [postgres]
    run_if_changed: ['^central/']
`)
	m, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, 1, m.Version)
	assert.Equal(t, []string{"go", "postgres"}, m.Jobs)
	assert.Equal(t, []string{"lint"}, m.AlwaysRun)
	require.Len(t, m.Rules, 2)
	assert.Equal(t, []string{"^ui/"}, m.Rules[0].SkipIfOnlyChanged)
	assert.Equal(t, []string{"^central/"}, m.Rules[1].RunIfChanged)
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"unknown field": "version: 1\njobs: [go]\nbogus: 1\n",
		"null value":    "version: 1\njobs: [go]\nalways_run:\n",
		"two documents": "version: 1\njobs: [go]\n---\nversion: 1\njobs: [x]\n",
	}
	for name, doc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Parse([]byte(doc))
			require.Error(t, err)
		})
	}
}

func TestCompileRejectsBadPattern(t *testing.T) {
	r := &Rule{Name: "bad", RunIfChanged: []string{"("}}
	require.Error(t, r.Compile())
}

func TestMatchHelpers(t *testing.T) {
	r := &Rule{
		Name:              "r",
		RunIfChanged:      []string{`^central/`},
		SkipIfOnlyChanged: []string{`^ui/`, `!^ui/keep\.txt$`},
	}
	require.NoError(t, r.Compile())

	assert.True(t, r.HasRun())
	assert.True(t, r.HasSkip())

	assert.True(t, r.RunMatchesAny([]string{"pkg/x.go", "central/y.go"}))
	assert.False(t, r.RunMatchesAny([]string{"pkg/x.go"}))

	assert.True(t, r.SkipCoversAll([]string{"ui/a.ts", "ui/b.ts"}))
	assert.False(t, r.SkipCoversAll([]string{"ui/a.ts", "pkg/x.go"}))
	// Empty diff is never covered.
	assert.False(t, r.SkipCoversAll(nil))
	// Exclusion removes the file from the skip set, so coverage breaks.
	assert.False(t, r.SkipCoversAll([]string{"ui/keep.txt"}))
}

func TestClausePresence(t *testing.T) {
	assert.False(t, (&Rule{}).HasRun())
	assert.False(t, (&Rule{}).HasSkip())
}
