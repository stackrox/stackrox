package main

// engines_test.go pins the wiring seams together: swapping the
// validator must change which findings stop selection, while the
// engine still fails closed on anything it cannot evaluate. These
// tests run without t.Parallel on purpose: they swap the rulesValidator
// wiring variable, and parallel tests must not observe the swap.

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/stackrox/rox/ci/select/internal/validation"
)

// swapValidator points the wiring at v for one test and restores it.
func swapValidator(t *testing.T, v validation.Validator) {
	t.Helper()
	previous := rulesValidator
	rulesValidator = v
	t.Cleanup(func() { rulesValidator = previous })
}

func TestTrivialValidatorDemonstratesTheSeam(t *testing.T) {
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

	swapValidator(t, validation.Trivial{})

	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"validate", "--rules", path}, &output, &diagnostic))
	assert.Equal(t, "Validation passed\n", output.String())

	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")
	output.Reset()
	diagnostic.Reset()
	err := run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic)
	require.ErrorIs(t, err, errLintReported)
	assert.Contains(t, diagnostic.String(), "compile patterns")
}

func TestValidatorSeamChangesRejectionsNotFailClosed(t *testing.T) {
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

	swapValidator(t, validation.Trivial{})

	files := writeTestFile(t, filepath.Join(t.TempDir(), "files"), "docs/guide.md\n")
	var output, diagnostic bytes.Buffer
	require.NoError(t, run(t.Context(), []string{"resolve", "--rules", path, "--files-file", files}, &output, &diagnostic))
	assert.Contains(t, output.String(), "go skip\n")
}
