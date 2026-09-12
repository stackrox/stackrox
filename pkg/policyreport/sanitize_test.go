package policyreport

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeProducerString(t *testing.T) {
	tests := map[string]struct {
		input    string
		expected string
	}{
		"empty string": {
			input:    "",
			expected: "",
		},
		"whitespace trimmed": {
			input:    "  hello  ",
			expected: "hello",
		},
		"control characters stripped": {
			input:    "hello\x00world\x07end",
			expected: "helloworldend",
		},
		"NUL stripped": {
			input:    "a\x00b",
			expected: "ab",
		},
		"BEL stripped": {
			input:    "a\x07b",
			expected: "ab",
		},
		"DEL stripped": {
			input:    "a\x7fb",
			expected: "ab",
		},
		"tabs preserved": {
			input:    "a\tb",
			expected: "a\tb",
		},
		"newlines stripped": {
			input:    "a\nb\rc",
			expected: "abc",
		},
		"under limit unchanged": {
			input:    "short string",
			expected: "short string",
		},
		"at limit unchanged": {
			input:    strings.Repeat("a", maxProducerStringLength),
			expected: strings.Repeat("a", maxProducerStringLength),
		},
		"over limit truncated": {
			input:    strings.Repeat("a", maxProducerStringLength+100),
			expected: strings.Repeat("a", maxProducerStringLength),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := sanitizeProducerString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeProducerString_UTF8Truncation(t *testing.T) {
	// 4-byte emoji at the truncation boundary must not be split.
	prefix := strings.Repeat("a", maxProducerStringLength-2)
	input := prefix + "\U0001F600" // 4-byte emoji, straddles byte 4094-4097

	result := sanitizeProducerString(input)

	assert.True(t, utf8.ValidString(result), "result must be valid UTF-8")
	assert.LessOrEqual(t, len(result), maxProducerStringLength)
	// The emoji doesn't fit within the byte limit, so it's excluded entirely.
	assert.Equal(t, prefix, result)
}

func TestSanitizeProducerString_MultiByteSafe(t *testing.T) {
	// CJK characters are 3 bytes each.
	input := strings.Repeat("世", maxProducerStringLength) // way over limit in bytes
	result := sanitizeProducerString(input)

	assert.True(t, utf8.ValidString(result), "result must be valid UTF-8")
	assert.LessOrEqual(t, len(result), maxProducerStringLength)
	// Every rune in the result should be complete.
	for i := 0; i < len(result); {
		r, size := utf8.DecodeRuneInString(result[i:])
		assert.NotEqual(t, utf8.RuneError, r, "no replacement characters allowed")
		i += size
	}
}
