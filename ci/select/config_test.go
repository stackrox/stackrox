package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const validRules = `version: 1
jobs: [go, postgres]
rules:
  - name: docs
    when: safe-paths
    paths: ['^docs/']
    skip: [go, postgres]
  - name: remaining
    when: remaining
    run: ['*']
`

func writeTestFile(t *testing.T, path, data string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(data), 0o600))
	return path
}

func TestLoadMapping(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"should reject malformed YAML":       "[",
		"should reject a second document":    validRules + "---\nversion: 1\n",
		"should reject empty input":          "",
		"should reject a scalar document":    "true",
		"should reject unsupported versions": strings.Replace(validRules, "version: 1", "version: 2", 1),
		"should reject an unknown field":     validRules + "typo: true\n",
		"should reject duplicate YAML keys":  validRules + "version: 1\n",
		"should reject empty inventory":      strings.Replace(validRules, "[go, postgres]", "[]", 1),
		"should reject duplicate jobs":       strings.Replace(validRules, "[go, postgres]", "[go, go]", 1),
		"should reject a wildcard job":       strings.Replace(validRules, "[go, postgres]", "['*']", 1),
		"should reject whitespace job names": strings.Replace(validRules, "[go, postgres]", "['go postgres']", 1),
		"should reject empty rules":          "version: 1\njobs: [go]\nrules: []\n",
		"should reject wildcard skips":       strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*']", 1),
		"should reject unknown skip jobs":    strings.Replace(validRules, "skip: [go, postgres]", "skip: [missing]", 1),
		"should reject repeated skip jobs":   strings.Replace(validRules, "skip: [go, postgres]", "skip: [go, go]", 1),
		"should reject mixed wildcard jobs":  strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*', go]", 1),
		"should reject repeated wildcard":    strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*', '*']", 1),
		"should reject no skip targets":      strings.Replace(validRules, "skip: [go, postgres]", "skip: []", 1),
		"should reject file rule runs":       strings.Replace(validRules, "skip: [go, postgres]", "run: ['*']", 1),
		"should reject bad patterns":         strings.Replace(validRules, "'^docs/'", "'^[broken'", 1),
		"should reject unrooted patterns":    strings.Replace(validRules, "'^docs/'", "'docs/'", 1),
		"should reject unrooted exclusions":  strings.Replace(validRules, "'^docs/'", "'^docs/', '!docs/testdata/'", 1),
		"should reject malformed exclusions": strings.Replace(validRules, "'^docs/'", "'^docs/', '!^[broken'", 1),
		"should reject empty exclusions":     strings.Replace(validRules, "'^docs/'", "'^docs/', '!'", 1),
		"should reject repeated negation":    strings.Replace(validRules, "'^docs/'", "'^docs/', '!!^docs/'", 1),
		"should reject exclusion-only rules": strings.Replace(validRules, "'^docs/'", "'!^docs/'", 1),
		"should reject no paths":             strings.Replace(validRules, "['^docs/']", "[]", 1),
		"should reject a missing name":       strings.Replace(validRules, "name: docs", "name: ''", 1),
		"should reject duplicate names":      strings.Replace(validRules, "name: docs", "name: remaining", 1),
		"should reject early remaining":      strings.Replace(validRules, "when: safe-paths", "when: remaining", 1),
		"should require final remaining":     strings.Replace(validRules, "when: remaining", "when: safe-paths", 1),
		"should require final run all":       strings.Replace(validRules, "run: ['*']", "run: [go]", 1),
		"should reject final wildcard skips": strings.Replace(validRules, "run: ['*']", "skip: ['*']", 1),
		"should reject final skips":          strings.Replace(validRules, "run: ['*']", "skip: [go, postgres]", 1),
		"should reject final paths":          strings.Replace(validRules, "run: ['*']", "run: ['*']\n    paths: ['^docs/']", 1),
		"should reject null fields":          strings.Replace(validRules, "skip: [go, postgres]", "skip: null", 1),
		"should reject null list elements":   strings.Replace(validRules, "'^docs/'", "null", 1),
		"should reject numeric jobs":         strings.Replace(validRules, "[go, postgres]", "[123]", 1),
		"should reject boolean paths":        strings.Replace(validRules, "'^docs/'", "true", 1),
		"should reject numeric names":        strings.Replace(validRules, "name: docs", "name: 123", 1),
		"should reject scalar targets":       strings.Replace(validRules, "skip: [go, postgres]", "skip: go", 1),
	}
	for _, field := range []string{"requires: {}", "label: example", "default: ['*']"} {
		cases["should reject removed field "+field] = strings.Replace(validRules, "    skip:", "    "+field+"\n    skip:", 1)
	}
	for _, when := range []string{"every-file-matches", "any-file-matches", "always", "no-file-changed", "label-exists", "typo"} {
		cases["should reject unused condition "+when] = strings.Replace(validRules, "safe-paths", when, 1)
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), data))
			require.Error(t, err)
		})
	}
	t.Run("should accept the supported schema", func(t *testing.T) {
		t.Parallel()
		m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), validRules))
		require.NoError(t, err)
		require.Len(t, m.Rules[0].patterns, 1)
	})
}
