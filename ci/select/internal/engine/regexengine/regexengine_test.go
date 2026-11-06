package regexengine

// regexengine_test.go pins the engine's own fail-closed line: Compile
// must reject every document it cannot evaluate faithfully, whatever
// the validator accepted, and Resolve must refuse foreign rules.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const baseRules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`

func TestCompileRejectsWhatItCannotEvaluate(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		data      string
		wantError string
	}{
		"should reject documents validation could have caught": {
			strings.Replace(baseRules, "version: 1", "version: 2", 1), "unsupported rules version 2, only 1 is supported",
		},
		"should reject malformed documents": {"[", "parse rules"},
		"should reject uncompilable patterns": {
			strings.Replace(baseRules, "'^docs/'", "'^[broken'", 1), "compile patterns",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := New().Compile([]byte(c.data))
			require.ErrorContains(t, err, c.wantError)
		})
	}
}

func TestResolveComputesDecisions(t *testing.T) {
	t.Parallel()
	compiled, err := New().Compile([]byte(baseRules))
	require.NoError(t, err)
	d := New().Resolve([]string{"docs/guide.md"}, compiled)
	assert.Equal(t, "skip", d.Action("go"))
	assert.Equal(t, "run", d.Action("unknown-job"))
}

// foreignRules implements engine.Rules without being this engine's
// representation; Resolve must fall back rather than guess.
type foreignRules struct{}

func TestResolveRefusesForeignRules(t *testing.T) {
	t.Parallel()
	d := New().Resolve(nil, foreignRules{})
	assert.Empty(t, d.Actions)
	assert.Contains(t, d.Log, "rules compiled by another engine")
}
