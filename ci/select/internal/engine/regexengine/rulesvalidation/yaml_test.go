package rulesvalidation

// yaml_test.go pins the middle depth: syntax only. It accepts documents
// the full checks reject and rejects documents no engine could read.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestYAMLChecksSyntaxOnly(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		data, wantError string
	}{
		"should accept a document the full checks reject": {"version: 2\njobs: [go]\nrules: []\n", ""},
		"should accept arbitrary yaml values":             {"- not\n- a\n- rules\n- file\n", ""},
		"should reject malformed yaml":                    {"[", "decode rules"},
		"should reject multiple documents":                {"version: 1\n---\nversion: 1\n", "single YAML document"},
		"should reject empty input":                       {"", "decode rules"},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			warnings, err := YAML{}.Validate([]byte(c.data))
			require.Empty(t, warnings)
			if c.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, c.wantError)
		})
	}
}
