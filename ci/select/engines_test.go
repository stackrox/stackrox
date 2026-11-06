package main

// engines_test.go pins the wiring: the engine supplies its own checks,
// so choosing a check depth changes which findings stop selection while
// the language still fails closed on anything it cannot evaluate. Each
// test builds its own engine value, so they run in parallel without
// shared state.

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/rox/ci/select/internal/engine/regexengine"
)

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
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", path}, &output, &diagnostic, regexengine.NewTrivial()))
	assert.Equal(t, "Validation passed\n", output.String())

	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")
	output.Reset()
	diagnostic.Reset()
	err := run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic, regexengine.NewTrivial())
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "compile patterns")
}

func TestCheckDepthChangesRejectionsNotFailClosed(t *testing.T) {
	t.Parallel()
	// Malformed YAML: syntax-only checks reject it before selection, and
	// with checks disabled the engine still refuses to compile it, so the
	// choice of depth never lets a bad document through.
	malformed := "version: 1\njobs: [go]\nrules: [\n"
	path := writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), malformed)
	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")

	var output, diagnostic bytes.Buffer
	err := run(t.Context(), []string{"validate", "--rules", path}, &output, &diagnostic, regexengine.New())
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "decode rules")

	output.Reset()
	diagnostic.Reset()
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", path}, &output, &diagnostic, regexengine.NewTrivial()))
	assert.Equal(t, "Validation passed\n", output.String())

	output.Reset()
	diagnostic.Reset()
	err = run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic, regexengine.NewTrivial())
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "parse rules")
}
