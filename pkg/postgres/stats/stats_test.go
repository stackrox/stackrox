package stats

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRedactQueryLiterals(t *testing.T) {
	cases := map[string]struct {
		input    string
		expected string
	}{
		"simple string literal": {
			input:    "SELECT * FROM users WHERE name = 'alice'",
			expected: "SELECT * FROM users WHERE name = $?",
		},
		"multiple literals": {
			input:    "SELECT * FROM t WHERE a = 'foo' AND b = 'bar'",
			expected: "SELECT * FROM t WHERE a = $? AND b = $?",
		},
		"escaped quote": {
			input:    "SELECT * FROM t WHERE name = 'it''s fine'",
			expected: "SELECT * FROM t WHERE name = $?",
		},
		"empty string literal": {
			input:    "SELECT * FROM t WHERE name = ''",
			expected: "SELECT * FROM t WHERE name = $?",
		},
		"parameterized query unchanged": {
			input:    "SELECT * FROM t WHERE id = $1 AND name = $2",
			expected: "SELECT * FROM t WHERE id = $1 AND name = $2",
		},
		"no literals": {
			input:    "SELECT count(*) FROM pg_stat_activity",
			expected: "SELECT count(*) FROM pg_stat_activity",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, redactQueryLiterals(tc.input))
		})
	}
}
