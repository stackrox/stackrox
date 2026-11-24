package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	data := []byte(`
jobs: [go, postgres, lint]
rules:
  - name: go-unit
    jobs: [go]
    skip_if_only_changed: ['^ui/']
  - name: db
    jobs: [postgres]
    run_if_changed: ['^central/']
  - name: lint
    jobs: [lint]
    always_run: true
`)
	m, err := Parse(data)
	require.NoError(t, err)
	assert.Equal(t, []JobEntry{{Name: "go"}, {Name: "postgres"}, {Name: "lint"}}, m.Jobs)
	require.Len(t, m.Rules, 3)
	assert.Equal(t, []string{"^ui/"}, m.Rules[0].SkipIfOnlyChanged)
	assert.Equal(t, []string{"^central/"}, m.Rules[1].RunIfChanged)
	assert.True(t, m.Rules[2].AlwaysRun)
}

func TestParseWithProwAliases(t *testing.T) {
	data := []byte(`
jobs:
  - go
  - name: e2e-tests
    prow_aliases: [gke-e2e-tests, ocp-e2e-tests]
  - lint
rules:
  - name: go-unit
    jobs: [go]
    always_run: true
  - name: e2e
    jobs: [e2e-tests]
    skip_if_only_changed: ['^.*\.md$']
  - name: lint-rule
    jobs: [lint]
    always_run: true
`)
	m, err := Parse(data)
	require.NoError(t, err)
	require.Len(t, m.Jobs, 3)
	assert.Equal(t, "e2e-tests", m.Jobs[1].Name)
	assert.Equal(t, []string{"gke-e2e-tests", "ocp-e2e-tests"}, m.Jobs[1].ProwAliases)

	assert.Equal(t, []string{"go", "e2e-tests", "lint"}, m.JobNames())
	assert.True(t, m.HasJob("go"))
	assert.False(t, m.HasJob("missing"))
	assert.Equal(t, "e2e-tests", m.ResolveByProwAlias("gke-e2e-tests"))
	assert.Equal(t, "e2e-tests", m.ResolveByProwAlias("ocp-e2e-tests"))
	assert.Equal(t, "", m.ResolveByProwAlias("unknown"))
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"unknown field": "jobs: [go]\nbogus: 1\n",
		"null value":    "jobs:\n",
		"two documents": "jobs: [go]\n---\njobs: [x]\n",
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
