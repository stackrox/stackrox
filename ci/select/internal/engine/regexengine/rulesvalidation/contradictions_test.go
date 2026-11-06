package rulesvalidation

// This suite pins the error half of the severity split: mutually
// exclusive statements no diff can satisfy. Overlapping claims that only
// some diffs would hit stay warnings and are pinned in conflicts_test.go.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestLoadRejectsDeadInclusion pins an inclusion a rule's own exclusion
// covers entirely. A dead run line and a dead skip line are both errors.
func TestFullRejectsDeadInclusion(t *testing.T) {
	t.Parallel()
	const deadInclusion = `version: 1
jobs: [go]
rules:
  - name: docs-with-dead-line
    when: any-file-matches
    paths:
      - '^docs/genuine\.md$'
      - '^docs/never/used\.md$'
      - '!^docs/never/'
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	for name, decision := range map[string]string{
		"should reject a dead skip line": "skip: [go]",
		"should reject a dead run line":  "run: [go]",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			rules := strings.Replace(deadInclusion, "skip: [go]", decision, 1)
			warnings, err := Full{}.Validate([]byte(rules))
			require.Error(t, err)
			require.ErrorContains(t, err, `rule "docs-with-dead-line": pattern "^docs/never/used\\.md$" matches only files its own exclusions cover`)
			require.ErrorContains(t, err, "e.g. \"docs/never/used.md\"")
			require.Contains(t, warnings, "Result: 1 error, 0 warnings. Errors fail the resolver.")
		})
	}
	// Without the dead line the same file is valid again.
	_, err := Full{}.Validate([]byte(strings.Replace(deadInclusion, "      - '^docs/never/used\\.md$'\n", "", 1)))
	require.NoError(t, err)
}

// TestLoadAcceptsAnchorThatMatchesNothing pins a later '^', which the
// production matcher rejects. It must not be reported as covering the
// files of a leading '^'.
func TestFullAcceptsAnchorThatMatchesNothing(t *testing.T) {
	t.Parallel()
	const live = `version: 1
jobs: [go]
rules:
  - name: almost
    when: any-file-matches
    paths: ['^foo', '!^foo^']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(live))
	require.NoError(t, err)
	require.Equal(t, []string{"Result: no issues found."}, warnings)
}

// TestUnmodeledCoveredRunStaysWarning pins a run pattern the automaton
// cannot model exactly. A word boundary is over-approximated, so coverage
// is not proof: the file loads with a warning and no error.
func TestUnmodeledCoveredRunStaysWarning(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/']
    skip: [go]
  - name: boundary
    when: any-file-matches
    paths: ['^docs/foo\b']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(rules))
	require.NoError(t, err)
	require.Contains(t, warnings, `001 Warning: "boundary" vs "docs" for "go"`)
	require.NotContains(t, strings.Join(warnings, "\n"), "e.g.")
	require.Contains(t, warnings, `Result: 0 errors, 1 warning. The "skip" wins all warning conflicts.`)
}

// The same include text on a run rule and a skip rule is an error when
// the two copies still match a common file. The skip covers the whole
// run, so the overlap warning is omitted and the covered-pattern error
// is not repeated.
func TestFullRejectsDuelingPins(t *testing.T) {
	t.Parallel()
	for name, pattern := range map[string]string{
		"exact file":   `^central/blah/readme\.md$`,
		"directory":    `^central/foo/`,
		"with symbols": `^docs/(one|two)\.md$`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			const duel = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['%s']
    skip: [go]
  - name: central-domain
    when: any-file-matches
    paths: ['%s']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
			_, err := Full{}.Validate([]byte(fmt.Sprintf(duel, pattern, pattern)))
			require.Error(t, err)
			require.ErrorContains(t, err, "both use")
			require.NotContains(t, err.Error(), "never runs")
		})
	}
	warnings, err := Full{}.Validate([]byte(`version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^central/blah/readme\.md$']
    skip: [go]
  - name: central-domain
    when: any-file-matches
    paths: ['^central/blah/readme\.md$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`))
	require.ErrorContains(t, err, `rules "central-domain" and "documentation" both use "^central/blah/readme\\.md$" for "go" and both match "central/blah/readme.md"`)
	require.NotContains(t, err.Error(), "never runs")
	require.Contains(t, warnings, "Result: 1 error, 0 warnings. Errors fail the resolver.")
}

// TestCoveredPatternKeepsThePartialWarning pins a run rule the skip does
// not cover entirely. README.md is the error; main.go does not match the
// skip rule, so the warning stays and its example is README.md.
func TestCoveredPatternKeepsThePartialWarning(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^(?is:.*\.md)$']
    skip: [go]
  - name: mixed
    when: any-file-matches
    paths: ['^README\.md$', '^main\.go$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(rules))
	require.Error(t, err)
	require.ErrorContains(t, err, `rule "mixed": pattern "^README\\.md$" never runs "go"`)
	require.NotContains(t, err.Error(), "main\\.go")
	require.Contains(t, warnings, `002 Warning: "mixed" vs "docs" for "go": both match e.g. "README.md"`)
	require.Contains(t, warnings, "Result: 1 error, 1 warning. The \"skip\" wins all warning conflicts. Errors fail the resolver.")
}

// Different pattern strings, with the run broader than the skip: some
// diffs hit the overlap and some do not, so this stays a warning.
func TestBroadCrossingWithPinStaysWarning(t *testing.T) {
	t.Parallel()
	const broad = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^central/blah/readme\.md$']
    skip: [go]
  - name: central-domain
    when: any-file-matches
    paths: ['^central/blah/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(broad))
	require.NoError(t, err, "a broad overlap is not a contradiction")
	require.Contains(t, warnings,
		`001 Warning: "central-domain" vs "documentation" for "go": both match e.g. "central/blah/readme.md"`)
}

// Two different operator patterns that happen to overlap: a crossing
// some diffs hit, so a warning, not a dueling pin.
func TestOperatorPatternIsNotAPin(t *testing.T) {
	t.Parallel()
	const fuzzy = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^docs/one\.md$']
    skip: [go]
  - name: shared-inputs
    when: any-file-matches
    paths: ['^docs/(one|two)\.md$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(fuzzy))
	require.NoError(t, err, "an operator pattern matches many files, so the crossing degrades to a warning")
	// Warning line, blank separator, footer.
	require.Len(t, warnings, 3)
}

// The same include text, with a carve-out on the skip side, still collides
// on the files the exclusion misses. The run still handles the carved-out
// files, so the message names the duplicated pattern rather than calling
// either rule dead.
func TestDuplicatedPatternWithCarveOutIsError(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/', '!^docs/internal/']
    skip: [go]
  - name: code
    when: any-file-matches
    paths: ['^docs/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(rules))
	require.Error(t, err)
	require.ErrorContains(t, err, `rules "code" and "docs" both use "^docs/" for "go" and both match "docs/"`)
	require.NotContains(t, err.Error(), "never runs")
	require.Contains(t, warnings, `002 Warning: "code" vs "docs" for "go": both match e.g. "docs/"`)
}

// A run pattern that matches only files one skip rule matches can never
// count. The strings differ, and the warning is omitted because no diff
// lets the run win.
func TestCoveredRunPatternIsError(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^(?is:.*\.md)$']
    skip: [go]
  - name: one-readme
    when: any-file-matches
    paths: ['^README\.md$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(rules))
	require.Error(t, err)
	require.ErrorContains(t, err, `rule "one-readme": pattern "^README\\.md$" never runs "go"; "docs" matches every file of that pattern (e.g. "README.md")`)
	require.Contains(t, warnings, "Result: 1 error, 0 warnings. Errors fail the resolver.")
}

// Identical text whose exclusions remove every shared file is not a
// duplicated-pattern error. The dead copy is its own error.
func TestExcludedDuplicateDoesNotCollide(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/', '!^docs/']
    skip: [go]
  - name: code
    when: any-file-matches
    paths: ['^docs/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	_, err := Full{}.Validate([]byte(rules))
	require.Error(t, err)
	require.ErrorContains(t, err, "matches only files its own exclusions cover")
	require.NotContains(t, err.Error(), "both use")
}

// TestLoadKeepsARunPastAnUnscopedDollar covers a '$' that binds to one
// alternative. fooX matches the run rule only, so the run claim is live.
func TestFullKeepsARunPastAnUnscopedDollar(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: run
    when: any-file-matches
    paths: ['^foo|bar$']
    run: [go]
  - name: skip
    when: any-file-matches
    paths: ['^(foo|bar)$']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	warnings, err := Full{}.Validate([]byte(rules))
	require.NoError(t, err)
	body := strings.Join(warnings, "\n")
	require.Contains(t, body, `"run" vs "skip" for "go"`)
	require.NotContains(t, body, "never runs")
}
