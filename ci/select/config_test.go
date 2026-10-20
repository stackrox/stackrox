package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadMappingYAML(t *testing.T) {
	const rules = `# Select changed Go files.
version: 1
jobs:
  - unit
rules:
  - name: go
    when: any-file-matches
    paths:
      - '^pkg/.*\.go$' # Preserve the literal backslash.
    run:
      - '*'
  - name: remaining
    when: remaining
    skip:
      - '*'
`
	m, err := loadMapping(fixturePath(t, rules))
	require.NoError(t, err)
	require.Len(t, m.Rules, 2)
	assert.Equal(t, []string{`^pkg/.*\.go$`}, m.Rules[0].Paths)
	assert.Equal(t, []string{"*"}, m.Rules[0].Run)
	assert.True(t, m.Rules[0].patterns[0].MatchString("pkg/example.go"))
	assert.False(t, m.Rules[0].patterns[0].MatchString("pkg/exampleXgo"))

	for name, test := range map[string]struct {
		text, errorText string
	}{
		"duplicate key": {
			text: strings.Replace(rules, "version: 1", "version: 1\nversion: 2", 1), errorText: "read YAML rules",
		},
		"boolean pattern": {
			text: strings.Replace(rules, "'^pkg/.*\\.go$'", "true", 1), errorText: "read YAML rules",
		},
		"null pattern": {
			text: strings.Replace(rules, "'^pkg/.*\\.go$'", "null", 1), errorText: "must not contain null values",
		},
		"unquoted wildcard": {
			text: strings.Replace(rules, "- '*'", "- *", 1), errorText: "read YAML rules",
		},
		"malformed document": {
			text: "version: [\n", errorText: "read YAML rules",
		},
		"multiple documents": {
			text: rules + "---\nversion: 2\n", errorText: "single YAML document",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := loadMapping(fixturePath(t, test.text))
			require.ErrorContains(t, err, test.errorText)
		})
	}
}
