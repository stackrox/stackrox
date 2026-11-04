package validation

// This suite pins Check's conflict report: every run/skip crossing is
// legal but warned about, with a witness file the overlap engine
// produced and the production matcher confirmed. Rejections (structure,
// targets, patterns) are pinned in load_test.go; how a crossing resolves
// at selection time is pinned in internal/resolver.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func writeRules(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rules.yaml")
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	return path
}

// Two rules with overlapping but differently written scope and opposite
// decisions: the crossing is warned about, not rejected — the skip
// default decides it.
const identicalScope = `version: 1
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

func TestCheckWarnsAboutConflict(t *testing.T) {
	t.Parallel()
	_, warnings, err := Load(writeRules(t, identicalScope))
	require.NoError(t, err, "conflicts are legal: the skip default decides them")
	require.Contains(t, warnings,
		`001 Warning: "special-input" vs "special-documentation" for "go": both match e.g. "config/special.txt"`)
	require.Contains(t, warnings,
		"Result: 0 errors, 1 warning. The \"skip\" wins all warning conflicts.")
}

// A path rule requesting every job is the run side of a conflict like any
// other; the wildcard branch of overlappingTargets attributes it to every
// job the skip rule lists.
func TestCheckWarnsAboutCoveredWildcardRun(t *testing.T) {
	t.Parallel()
	const coveredWildcard = `version: 1
jobs: [go, postgres]
rules:
  - name: force-on-config
    when: any-file-matches
    paths: ['^config/']
    run: ['*']
  - name: documentation
    when: any-file-matches
    paths: ['^config/.*\.md$']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	_, warnings, err := Load(writeRules(t, coveredWildcard))
	require.NoError(t, err)
	require.Contains(t, warnings,
		`001 Warning: "force-on-config" vs "documentation" for "go": both match e.g. "config/.md"`)
}

// A run pattern the skip covers entirely can never count. That is an
// error, and the overlap warning is omitted because no diff lets the
// run win.
func TestLoadRejectsRunPatternTheSkipCovers(t *testing.T) {
	t.Parallel()
	const covered = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^.*\.txt$']
    skip: [go]
  - name: generator-header
    when: any-file-matches
    paths: ['^tools/boilerplate\.go\.txt$']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	_, warnings, err := Load(writeRules(t, covered))
	require.Error(t, err)
	require.ErrorContains(t, err, `rule "generator-header": pattern "^tools/boilerplate\\.go\\.txt$" never runs "go"; "documentation" matches every file of that pattern (e.g. "tools/boilerplate.go.txt")`)
	require.Contains(t, warnings, "Result: 1 error, 0 warnings. Errors fail the resolver.")
}

// Disjoint rules produce no conflict warnings at all.
func TestCheckStaysSilentWithoutConflicts(t *testing.T) {
	t.Parallel()
	const disjoint = `version: 1
jobs: [go]
rules:
  - name: documentation
    when: any-file-matches
    paths: ['^docs/']
    skip: [go]
  - name: scanner-inputs
    when: any-file-matches
    paths: ['^scanner/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	_, warnings, err := Load(writeRules(t, disjoint))
	require.NoError(t, err)
	// The footer is "no issues found" even on the empty report, so the
	// warnings slice holds exactly that line.
	require.Len(t, warnings, 1)
	require.Equal(t, "Result: no issues found.", warnings[0])
}

// An end anchor wrapped by the closing group is still an end anchor.
// docs/*.md and scanner/ share no file, so the pair stays silent.
func TestEndAnchorInsideGroupStaysDisjoint(t *testing.T) {
	t.Parallel()
	const rules = `version: 1
jobs: [go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^(?is:docs/.*\.md$)']
    skip: [go]
  - name: scanner
    when: any-file-matches
    paths: ['^scanner/']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	_, warnings, err := Load(writeRules(t, rules))
	require.NoError(t, err)
	require.Equal(t, []string{"Result: no issues found."}, warnings)
}
