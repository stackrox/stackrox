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

func TestCompileRejectsInvalidStructure(t *testing.T) {
	t.Parallel()
	for name, data := range map[string]string{
		"should reject empty jobs":                strings.Replace(baseRules, "jobs: [go]", "jobs: []", 1),
		"should reject duplicate jobs":            strings.Replace(baseRules, "jobs: [go]", "jobs: [go, go]", 1),
		"should reject wildcard jobs":             strings.Replace(baseRules, "jobs: [go]", "jobs: ['*']", 1),
		"should reject empty job names":           strings.Replace(baseRules, "jobs: [go]", "jobs: ['']", 1),
		"should reject whitespace in job names":   strings.Replace(baseRules, "jobs: [go]", "jobs: ['go tests']", 1),
		"should reject comment markers in jobs":   strings.Replace(baseRules, "jobs: [go]", "jobs: ['go#tests']", 1),
		"should reject empty rules":               "version: 1\njobs: [go]\nrules: []\n",
		"should reject empty rule names":          strings.Replace(baseRules, "name: docs", "name: ''", 1),
		"should reject duplicate rule names":      strings.Replace(baseRules, "name: remaining", "name: docs", 1),
		"should reject unknown conditions":        strings.Replace(baseRules, "when: any-file-matches", "when: any-files-match", 1),
		"should reject missing paths":             strings.Replace(baseRules, "paths: ['^docs/']", "paths: []", 1),
		"should reject missing targets":           strings.Replace(baseRules, "skip: [go]", "skip: []", 1),
		"should reject unknown targets":           strings.Replace(baseRules, "skip: [go]", "skip: [typo]", 1),
		"should reject repeated targets":          strings.Replace(baseRules, "skip: [go]", "skip: [go, go]", 1),
		"should reject opposing targets":          strings.Replace(baseRules, "skip: [go]", "skip: [go]\n    run: [go]", 1),
		"should reject wildcard skips":            strings.Replace(baseRules, "skip: [go]", "skip: ['*']", 1),
		"should reject mixed wildcard targets":    strings.Replace(baseRules, "skip: [go]", "run: ['*', go]", 1),
		"should reject wildcard runs with skips":  strings.Replace(baseRules, "skip: [go]", "skip: [go]\n    run: ['*']", 1),
		"should reject empty patterns":            strings.Replace(baseRules, "'^docs/'", "''", 1),
		"should reject unrooted patterns":         strings.Replace(baseRules, "'^docs/'", "'docs/'", 1),
		"should reject unrooted exclusions":       strings.Replace(baseRules, "'^docs/'", "'^docs/', '!testdata/'", 1),
		"should reject exclusion-only paths":      strings.Replace(baseRules, "'^docs/'", "'!^docs/'", 1),
		"should reject missing remaining rule":    strings.Split(baseRules, "  - name: remaining")[0],
		"should reject misplaced remaining rule":  strings.Replace(baseRules, "when: any-file-matches", "when: remaining", 1),
		"should reject paths on remaining rule":   baseRules + "    paths: ['^docs/']\n",
		"should reject skips on remaining rule":   baseRules + "    skip: [go]\n",
		"should reject partial remaining targets": strings.Replace(baseRules, "run: ['*']", "run: [go]", 1),
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			for depth, eng := range map[string]engine.Engine{"yaml": New(), "trivial": NewTrivial()} {
				t.Run(depth, func(t *testing.T) {
					t.Parallel()
					compiled, err := eng.Compile([]byte(data))
					require.Error(t, err)
					assert.Nil(t, compiled)
				})
			}
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
