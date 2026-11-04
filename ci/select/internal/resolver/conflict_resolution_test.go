package resolver

// This suite pins how a conflict between a run request and a skip claim on
// the same file resolves. A run request counts only for a file the job's
// skip rules do not cover, so on a shared file the skip wins:
//
//   - TestSkipWinsOnFileSpecific: a skip specific to the file (a README)
//     wins over a domain-wide run request.
//   - TestRunWinsThroughCarveOut: a '!' exclusion lets the run count.
//     Dropping it leaves a run pattern the skip covers entirely, which
//     is an error.
//   - TestSkipWinsOnIdenticalScope: the same scope with opposite decisions
//     ends in skip; one of the rules must change.
//
// A crossing only some files hit is a warning. A run pattern one skip
// rule covers entirely is an error. See README.md ("Composition of rules").

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackrox/rox/ci/select/internal/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// conflictSkipSpecific builds the conflict where the skip is more specific
// than the run: central-domain requests go for every file under ^central/sac/,
// while documentation covers exactly README files, so central/sac/README.md
// matches both and the file-specific skip wins. A .go file in the same
// directory matches only the domain rule, so its run request counts.
const conflictSkipSpecific = `version: 1
jobs: [go, sensor-integration]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^(.*/)?README\.md$']
    skip: [go, sensor-integration]
  - name: central-domain
    when: any-file-matches
    paths: ['^central/sac/']
    run: [go]
    skip: [sensor-integration]
  - name: remaining
    when: remaining
    run: ['*']
`

func loadValidatedRules(t *testing.T, data string) rules.Mapping {
	t.Helper()
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), data)
	// loadRules is the production loader: a fixture is checked fully —
	// structure and conflicts — like a real rules file, so a fixture
	// typo cannot silently pass.
	m, err := loadRules(path)
	require.NoError(t, err)
	return m
}

func TestSkipWinsOnFileSpecificConflict(t *testing.T) {
	t.Parallel()
	// Central-domain runs go; documentation covers the README, so the
	// skip default settles the crossing.
	m := loadValidatedRules(t, conflictSkipSpecific)

	t.Run("a lone domain README should skip the requested job", func(t *testing.T) {
		t.Parallel()
		d := Resolve([]string{"central/sac/README.md"}, m)
		assert.Equal(t, "skip", d.Action("go"))
		// The trace names both sides of the crossing and the outcome.
		assert.Contains(t, d.Log,
			`conflict on path 1 (central/sac/README.md): "central-domain" requests run, "documentation" marks it safe to skip, so the skip wins`)
	})

	t.Run("a code file in the same domain should still run it", func(t *testing.T) {
		t.Parallel()
		d := Resolve([]string{"central/sac/README.md", "central/sac/service.go"}, m)
		assert.Equal(t, "run", d.Action("go"))
		assert.Contains(t, d.Log, `run requested by "central-domain" for paths: 2`)
		assert.Equal(t, "skip", d.Action("sensor-integration"))
	})
}

func TestRunWinsThroughCarveOut(t *testing.T) {
	t.Parallel()
	// generator-header includes the header file and documentation excludes it,
	// so the run request counts. Dropping that exclusion makes documentation
	// cover every file of the run pattern, which validation rejects.
	const withCarveOut = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths:
      - '^.*\.txt$'
      - '!^tools/boilerplate\.go\.txt$'
    skip: [go]
  - name: generator-header
    when: any-file-matches
    paths: ['^tools/boilerplate\.go\.txt$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`

	t.Run("go should run when the skip rule excludes the real input", func(t *testing.T) {
		t.Parallel()
		m := loadValidatedRules(t, withCarveOut)
		d := Resolve([]string{"tools/boilerplate.go.txt"}, m)
		// No coverage, so the run request counts and no conflict
		// exists.
		assert.Equal(t, "run", d.Action("go"))
		assert.NotContains(t, d.Log, "conflict on path")
		assert.Contains(t, d.Log, `"tools/boilerplate.go.txt" matched: generator-header`)
	})

	t.Run("dropping the carve-out is an error", func(t *testing.T) {
		t.Parallel()
		// Without the exclusion, documentation matches every file of the
		// run pattern, so the request can never count.
		withoutCarveOut := strings.Replace(withCarveOut, "      - '!^tools/boilerplate\\.go\\.txt$'\n", "", 1)
		path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), withoutCarveOut)
		_, err := loadRules(path)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `never runs "go"`)
		assert.Contains(t, err.Error(), "documentation")
	})
}

// Two conflicting rules with opposite decisions over the same files, via
// differently written but overlapping scope: the skip default settles it
// in favor of the skip, and the trace names the crossing so the coupling
// is visible when it happens. (The identical pattern string in both rules
// would be rejected as a dueling pin by validation.)
const conflictIdenticalScope = `version: 1
jobs: [go]
rules:
  - name: special-input
    when: any-file-matches
    paths: ['^config/.*\.txt$']
    run: [go]
  - name: special-documentation
    when: any-file-matches
    paths: ['^config/special\.txt$']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`

func TestSkipWinsOnIdenticalScope(t *testing.T) {
	t.Parallel()
	m := loadValidatedRules(t, conflictIdenticalScope)
	d := Resolve([]string{"config/special.txt"}, m)
	assert.Equal(t, "skip", d.Action("go"))
	assert.Contains(t, d.Log,
		`conflict on path 1 (config/special.txt): "special-input" requests run, "special-documentation" marks it safe to skip, so the skip wins`)
}

// TestRealInputsHaveNoDocumentationCoverage keeps the principle honest for
// the current repository: every known embedded or generated input with a
// documentation extension should be carved out of the documentation rule,
// so real input files never lose their run requests to the skip default.
func TestRealInputsHaveNoDocumentationCoverage(t *testing.T) {
	t.Parallel()
	m, err := loadRules("../../../test-selection-rules.yaml")
	require.NoError(t, err)
	for _, path := range []string{
		// Read by the code generators (make generated-srcs).
		"tools/boilerplate.go.txt",
		"operator/hack/boilerplate.go.txt",
		"config-controller/hack/boilerplate.go.txt",
		// Images downloaded by Sensor and e2e tests.
		"sensor/tests/images-to-prefetch.txt",
		"tests/images-to-prefetch.txt",
		"qa-tests-backend/scripts/images-to-prefetch.txt",
		// Python dependencies for scripts that CI executes.
		".openshift-ci/dev-requirements.txt",
		"tests/performance/scripts/opensearch-upload/requirements.txt",
	} {
		t.Run(path, func(t *testing.T) {
			t.Parallel()
			d := Resolve([]string{path}, m)
			assert.Equal(t, "run", d.Action("go"), "the documentation rule should not cover real inputs")
			assert.NotContains(t, d.Log, `covered by: "documentation-only"`, path)
		})
	}
}
