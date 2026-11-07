package regexengine

// regexengine_test.go pins the engine's own fail-closed line: Compile
// must reject every document it cannot evaluate faithfully, whatever
// its checks accepted, and Resolve must refuse foreign rules. It also
// pins that the language hands out its own checks, at two depths.

import (
	"strings"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/engine"
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

func TestValidateUsesThisLanguageChecks(t *testing.T) {
	t.Parallel()
	malformed := "version: 1\njobs: [go]\nrules: [\n"
	for name, c := range map[string]struct {
		engine    func() engine.Engine
		wantError string
	}{
		"should check yaml syntax by default":        {New, "decode rules"},
		"should disable authoring checks on request": {NewTrivial, ""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := c.engine().Validate([]byte(malformed))
			if c.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, c.wantError)
		})
	}
}

// TestAcceptedDocumentsAlwaysCompile pins the ordering the wiring
// relies on: whatever this language's default checks accept, Compile can
// evaluate, so validation can never wave through a document the engine
// then refuses to interpret.
func TestAcceptedDocumentsAlwaysCompile(t *testing.T) {
	t.Parallel()
	e := New()
	for name, data := range map[string]string{
		"minimal policy": baseRules,
		"exclusions":     strings.Replace(baseRules, "'^docs/'", "'^docs/', '!^docs/testdata/'", 1),
		"wildcard run":   strings.Replace(baseRules, "skip: [go]", "run: ['*']", 1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := e.Validate([]byte(data))
			require.NoError(t, err, "fixture must pass validation")
			_, err = e.Compile([]byte(data))
			require.NoError(t, err, "accepted documents must compile")
		})
	}
}
