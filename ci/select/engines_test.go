package main

// engines_test.go pins the wiring: the engine supplies its own checks,
// so choosing a check depth changes which findings stop selection while
// the language still fails closed on anything it cannot evaluate. Each
// test builds its own deps value, so they run in parallel without
// shared state.

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
)

// trivialDeps is today's language with its authoring checks disabled.
func trivialDeps() deps {
	return deps{engine: regexengine.NewTrivial()}
}

func TestTrivialValidatorDemonstratesTheSeam(t *testing.T) {
	t.Parallel()
	// A document Full rejects for a broken pattern: with Trivial it
	// passes the lint step, and the engine's own Compile still fails
	// closed with the same finding class.
	broken := `version: 1
jobs: [go]
rules:
  - name: bad
    when: any-file-matches
    paths: ['^[broken']
    run: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), broken)

	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", path}, &output, &diagnostic, trivialDeps()))
	assert.Equal(t, "Validation passed\n", output.String())

	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")
	output.Reset()
	diagnostic.Reset()
	err := run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic, trivialDeps())
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "compile patterns")
}

func TestValidatorSeamChangesRejectionsNotFailClosed(t *testing.T) {
	t.Parallel()
	// A document Full rejects only for authoring quality (duplicate
	// targets): swapping to Trivial lets it resolve, proving the
	// validator choice changes which findings stop selection while the
	// engine keeps deciding normally.
	loose := `version: 1
jobs: [go, go]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/']
    skip: [go]
  - name: remaining
    when: remaining
    run: ['*']
`
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), loose)
	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")

	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic, trivialDeps()))
	assert.Contains(t, output.String(), "go skip\n")

	output.Reset()
	diagnostic.Reset()
	err := run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic, productionDeps())
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "duplicate job")
}
