package engine

// validator_test.go pins TrivialValidator's end of the bargain: it
// reports nothing for any document, so a language that offers it lowers
// only authoring checks. Compilation remains the gate that keeps
// selection fail-closed.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrivialValidatorAcceptsEveryDocument(t *testing.T) {
	t.Parallel()
	for name, data := range map[string]string{
		"should accept documents another language accepts": "version: 2\njobs: []\nrules: []\n",
		"should accept non-yaml documents":                 "[",
		"should accept empty documents":                    "",
		"should accept uncompilable patterns":              "version: 1\njobs: [go]\nrules:\n  - name: bad\n    when: any-file-matches\n    paths: ['^[broken']\n    run: [go]\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			warnings, err := TrivialValidator{}.Validate([]byte(data))
			require.NoError(t, err)
			require.Empty(t, warnings)
		})
	}
}
