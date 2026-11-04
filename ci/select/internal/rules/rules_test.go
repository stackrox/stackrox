package rules

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validRules = `version: 1
jobs: [go, postgres]
rules:
  - name: docs
    when: any-file-matches
    paths: ['^docs/']
    skip: [go, postgres]
  - name: remaining
    when: remaining
    run: ['*']
`

// Parse rejects malformed documents — bad YAML, a second document, nulls,
// wrong scalar types, unknown fields — but no rule substance; those
// cases live in the validation package.
func TestParse(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"should reject malformed YAML":      "[",
		"should reject a second document":   validRules + "---\nversion: 1\n",
		"should reject empty input":         "",
		"should reject a scalar document":   "true",
		"should reject an unknown field":    validRules + "typo: true\n",
		"should reject duplicate YAML keys": validRules + "version: 1\n",
		"should reject null fields":         strings.Replace(validRules, "skip: [go, postgres]", "skip: null", 1),
		"should reject null list elements":  strings.Replace(validRules, "'^docs/'", "null", 1),
		"should reject numeric jobs":        strings.Replace(validRules, "[go, postgres]", "[123]", 1),
		"should reject boolean paths":       strings.Replace(validRules, "'^docs/'", "true", 1),
		"should reject numeric names":       strings.Replace(validRules, "name: docs", "name: 123", 1),
		"should reject scalar targets":      strings.Replace(validRules, "skip: [go, postgres]", "skip: go", 1),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(data))
			require.Error(t, err)
		})
	}
	t.Run("should decode the minimal document", func(t *testing.T) {
		t.Parallel()
		m, err := Parse([]byte(validRules))
		require.NoError(t, err)
		require.Equal(t, 1, m.Version)
		require.Equal(t, []string{"go", "postgres"}, m.Jobs)
		require.Len(t, m.Rules, 2)
		require.Equal(t, "docs", m.Rules[0].Name)
	})
}

func TestCompile(t *testing.T) {
	t.Parallel()
	t.Run("should split inclusions from exclusions", func(t *testing.T) {
		t.Parallel()
		m, err := Parse([]byte(strings.Replace(validRules, "'^docs/'", "'^docs/', '!^docs/testdata/'", 1)))
		require.NoError(t, err)
		require.NoError(t, m.Rules[0].Compile())
		require.Len(t, m.Rules[0].patterns, 1)
		require.Len(t, m.Rules[0].excludedPatterns, 1)
	})
	t.Run("should reject a pattern that does not compile", func(t *testing.T) {
		t.Parallel()
		m, err := Parse([]byte(strings.Replace(validRules, "'^docs/'", "'^[broken'", 1)))
		require.NoError(t, err)
		err = m.Rules[0].Compile()
		require.ErrorContains(t, err, `rule "docs" pattern "^[broken"`)
	})
}
