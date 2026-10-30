package main

// This suite documents how rule conflicts resolve when one rule requests a
// run for a job and another rule marks the same file safe to skip that job.
// The resolution looks inconsistent at first glance — sometimes the skip
// wins, sometimes the effective outcome is a run — so the reasoning is
// spelled out here once, and pinned by the cases below.
//
// The principle: a run request counts only for a file that the job's skip
// rules do not already cover. Conflicts are therefore never resolved by
// ranking rules; they are resolved by the accuracy of the skip coverage.
// Coverage is a positive, reviewed claim about exact files ("this file
// cannot affect the job"), while a run request may be a coarse directory
// proxy ("things in this area tend to matter"). When both match one file:
//
//   - If the coverage is legitimately specific to the file (documentation
//     identified by its kind), the skip wins: TestSkipWinsOnFileSpecific
//     covers the domain-README shape, which is why central/sac/README.md
//     can skip go even though a domain rule requests it.
//   - If the file is actually a program input (a run rule names the real
//     input), the skip rule must not claim it in the first place. The
//     carve-out exclusion — not the conflict tie-break — is the supported
//     way to make the run win: TestRunWinsThroughCarveOut shows the
//     boilerplate.go.txt shape, where the documentation rule excludes a
//     generator header so the run request survives. Its companion case,
//     run without the carve-out, documents the trap: the skip would win
//     and silently skip tests on a real input. That is why carve-outs for
//     known inputs are mandatory, and why a conflict bullet in a trace
//     means "a skip rule is claiming an input it should not" — the fix is
//     always to narrow the skip with an exclusion, which also protects
//     every other file of that kind the run rule does not name.
//
// See the "Composition of rules" section in README.md and the walkthroughs
// in EXAMPLES.md for the user-facing description of the same behavior.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func loadConflictRules(t *testing.T, data string) mapping {
	t.Helper()
	m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), data))
	require.NoError(t, err)
	return m
}

func TestSkipWinsOnFileSpecificConflict(t *testing.T) {
	t.Parallel()
	m := loadConflictRules(t, conflictSkipSpecific)

	t.Run("and a lone domain README skips the requested job", func(t *testing.T) {
		t.Parallel()
		d := resolve([]string{"central/sac/README.md"}, m)
		assert.Equal(t, "skip", d.action("go"))
		assert.Contains(t, d.log, `conflict on path 1 (central/sac/README.md): ["central-domain"] requests run`)
		assert.Contains(t, d.log, `["documentation"] marks it safe to skip; the skip wins because it is specific to this file`)
	})

	t.Run("while a code file in the same domain still runs it", func(t *testing.T) {
		t.Parallel()
		d := resolve([]string{"central/sac/README.md", "central/sac/service.go"}, m)
		assert.Equal(t, "run", d.action("go"))
		assert.Contains(t, d.log, `run requested by ["central-domain"] for paths: 2`)
		assert.Equal(t, "skip", d.action("sensor-integration"))
	})
}

func TestRunWinsThroughCarveOut(t *testing.T) {
	t.Parallel()
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

	t.Run("when the skip rule excludes the real input", func(t *testing.T) {
		t.Parallel()
		m := loadConflictRules(t, withCarveOut)
		d := resolve([]string{"tools/boilerplate.go.txt"}, m)
		// No coverage, so the run request survives and no conflict exists.
		assert.Equal(t, "run", d.action("go"))
		assert.NotContains(t, d.log, "conflict on path")
		assert.Contains(t, d.log, `"tools/boilerplate.go.txt" matched: generator-header`)
	})

	t.Run("and the carve-out is required because without it the skip would win", func(t *testing.T) {
		t.Parallel()
		// Dropping only the exclusion documents the trap the header comment
		// describes: coverage claims an actual input, the tie-break trusts
		// the coverage, and a generator-header change would silently skip.
		withoutCarveOut := strings.Replace(withCarveOut, "      - '!^tools/boilerplate\\.go\\.txt$'\n", "", 1)
		m := loadConflictRules(t, withoutCarveOut)
		d := resolve([]string{"tools/boilerplate.go.txt"}, m)
		assert.Equal(t, "skip", d.action("go"))
		assert.Contains(t, d.log, `conflict on path 1 (tools/boilerplate.go.txt): ["generator-header"] requests run`)
		assert.Contains(t, d.log, `["documentation"] marks it safe to skip`)
	})
}

// TestRealInputsHaveNoDocumentationCoverage keeps the principle honest for
// the current repository: every known embedded or generated input with a
// documentation extension must be carved out of the documentation rule, so
// the "run wins" direction never depends on the tie-break for real files.
func TestRealInputsHaveNoDocumentationCoverage(t *testing.T) {
	t.Parallel()
	m, err := loadMapping("../test-selection-rules.yaml")
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
			d := resolve([]string{path}, m)
			assert.Equal(t, "run", d.action("go"), "documentation rule must not cover real inputs")
			assert.NotContains(t, d.log, `covered by: ["documentation-only"]`, path)
		})
	}
}
