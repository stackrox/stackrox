package validation

// validator_test.go pins the neutral end of the seam: Trivial must hold
// its part of the fail-closed bargain — an empty report, never a wrong
// one — so the engine's own checks remain the only gate it can lower.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrivialAcceptsEveryDocument(t *testing.T) {
	t.Parallel()
	for name, data := range map[string]string{
		"should accept documents full validation rejects":   "version: 2\njobs: []\nrules: []\n",
		"should accept non-yaml documents":                  "[",
		"should accept empty documents":                     "",
		"should accept documents the engine cannot compile": "version: 1\njobs: [go]\nrules:\n  - name: bad\n    when: any-file-matches\n    paths: ['^[broken'\n    run: [go]\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			warnings, err := Trivial{}.Validate([]byte(data))
			require.NoError(t, err)
			require.Empty(t, warnings)
		})
	}
}
