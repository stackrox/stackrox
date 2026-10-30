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
    when: any-file-matches
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
		"should reject padded job names":     strings.Replace(validRules, "[go, postgres]", "[' postgres']", 1),
		"should reject job names with hash":  strings.Replace(validRules, "[go, postgres]", "['pos#tgres']", 1),
		"should reject empty rules":          "version: 1\njobs: [go]\nrules: []\n",
		"should reject wildcard skips":       strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*']", 1),
		"should reject unknown skip jobs":    strings.Replace(validRules, "skip: [go, postgres]", "skip: [missing]", 1),
		"should reject repeated skip jobs":   strings.Replace(validRules, "skip: [go, postgres]", "skip: [go, go]", 1),
		"should reject mixed wildcard jobs":  strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*', go]", 1),
		"should reject repeated wildcard":    strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*', '*']", 1),
		"should reject no actions":           strings.Replace(validRules, "skip: [go, postgres]", "skip: []", 1),
		"should reject unknown run jobs":     strings.Replace(validRules, "skip: [go, postgres]", "run: [missing]", 1),
		"should reject repeated run jobs":    strings.Replace(validRules, "skip: [go, postgres]", "run: [go, go]", 1),
		"should reject mixed wildcard runs":  strings.Replace(validRules, "skip: [go, postgres]", "run: ['*', go]", 1),
		"should reject repeated run stars":   strings.Replace(validRules, "skip: [go, postgres]", "run: ['*', '*']", 1),
		"should reject conflicting actions":  strings.Replace(validRules, "skip: [go, postgres]", "run: [go]\n    skip: [go]", 1),
		"should reject conflicting wildcard": strings.Replace(validRules, "skip: [go, postgres]", "run: ['*']\n    skip: [go]", 1),
		"should reject scalar run targets":   strings.Replace(validRules, "skip: [go, postgres]", "run: go", 1),
		"should reject bad patterns":         strings.Replace(validRules, "'^docs/'", "'^[broken'", 1),
		"should reject unbalanced groups":    strings.Replace(validRules, "'^docs/'", "'^ui/)|docs/(?:'", 1),
		"should reject unrooted patterns":    strings.Replace(validRules, "'^docs/'", "'docs/'", 1),
		"should reject unrooted exclusions":  strings.Replace(validRules, "'^docs/'", "'^docs/', '!docs/testdata/'", 1),
		"should reject malformed exclusions": strings.Replace(validRules, "'^docs/'", "'^docs/', '!^[broken'", 1),
		"should reject empty exclusions":     strings.Replace(validRules, "'^docs/'", "'^docs/', '!'", 1),
		"should reject repeated negation":    strings.Replace(validRules, "'^docs/'", "'^docs/', '!!^docs/'", 1),
		"should reject exclusion-only rules": strings.Replace(validRules, "'^docs/'", "'!^docs/'", 1),
		"should reject no paths":             strings.Replace(validRules, "['^docs/']", "[]", 1),
		"should reject a missing name":       strings.Replace(validRules, "name: docs", "name: ''", 1),
		"should reject duplicate names":      strings.Replace(validRules, "name: docs", "name: remaining", 1),
		"should reject early remaining":      strings.Replace(validRules, "when: any-file-matches", "when: remaining", 1),
		"should require final remaining":     strings.Replace(validRules, "when: remaining", "when: any-file-matches", 1),
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
	for _, when := range []string{"safe-paths", "changed-paths", "every-file-matches", "always", "no-file-changed", "label-exists", "typo"} {
		cases["should reject unused condition "+when] = strings.Replace(validRules, "any-file-matches", when, 1)
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), data))
			require.Error(t, err)
		})
	}
	for name, actions := range map[string]string{
		"should accept skip targets":     "skip: [go, postgres]",
		"should accept run targets":      "run: [go, postgres]",
		"should accept all run targets":  "run: ['*']",
		"should accept disjoint actions": "run: [go]\n    skip: [postgres]",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data := strings.Replace(validRules, "skip: [go, postgres]", actions, 1)
			m, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), data))
			require.NoError(t, err)
			require.Len(t, m.Rules[0].patterns, 1)
		})
	}
}

func TestLoadMappingErrors(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		data, wantError string
	}{
		"unsupported version": {strings.Replace(validRules, "version: 1", "version: 2", 1), "unsupported rules version 2"},
		"missing jobs list":   {"version: 1\nrules: []\n", "nonempty jobs list"},
		"missing rules list":  {"version: 1\njobs: [go]\nrules: []\n", "nonempty rules list"},
		"unknown target":      {strings.Replace(validRules, "skip: [go, postgres]", "skip: [missing]", 1), "names unknown target \"missing\""},
		"wildcard in skip":    {strings.Replace(validRules, "skip: [go, postgres]", "skip: ['*']", 1), "the wildcard is supported only as run: ['*']"},
		"mixed run targets":   {strings.Replace(validRules, "skip: [go, postgres]", "run: ['*', go]", 1), "the wildcard is supported only as run: ['*']"},
		"repeated target":     {strings.Replace(validRules, "skip: [go, postgres]", "run: [go]\n    skip: [go]", 1), "repeats target \"go\""},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := loadMapping(writeTestFile(t, filepath.Join(t.TempDir(), "rules.yaml"), c.data))
			require.ErrorContains(t, err, c.wantError)
		})
	}
}
