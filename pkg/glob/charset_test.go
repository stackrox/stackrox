package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRuneRangeNormalise(t *testing.T) {
	tests := map[string]struct {
		input    []runeRange
		negated  bool
		expected []runeRange
	}{
		"empty": {
			input:    nil,
			expected: nil,
		},
		"single range": {
			input:    []runeRange{{Lo: 'a', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'z'}},
		},
		"non-overlapping sorted": {
			input:    []runeRange{{Lo: 'a', Hi: 'c'}, {Lo: 'x', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'c'}, {Lo: 'x', Hi: 'z'}},
		},
		"overlapping merged": {
			input:    []runeRange{{Lo: 'a', Hi: 'm'}, {Lo: 'k', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'z'}},
		},
		"adjacent merged": {
			input:    []runeRange{{Lo: 'a', Hi: 'c'}, {Lo: 'd', Hi: 'f'}},
			expected: []runeRange{{Lo: 'a', Hi: 'f'}},
		},
		"unsorted": {
			input:    []runeRange{{Lo: 'x', Hi: 'z'}, {Lo: 'a', Hi: 'c'}},
			expected: []runeRange{{Lo: 'a', Hi: 'c'}, {Lo: 'x', Hi: 'z'}},
		},
		"three overlapping": {
			input:    []runeRange{{Lo: 'a', Hi: 'e'}, {Lo: 'c', Hi: 'h'}, {Lo: 'g', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'z'}},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			cs := charSet{Ranges: tc.input, Negated: tc.negated}
			cs.normalize()
			assert.Equal(t, tc.expected, cs.Ranges)
		})
	}
}

func TestCharSetIsEmpty(t *testing.T) {
	tests := map[string]struct {
		cs       charSet
		expected bool
	}{
		"positive empty": {
			cs:       charSet{},
			expected: true,
		},
		"positive non-empty": {
			cs:       singleChar('a'),
			expected: false,
		},
		"negated empty ranges (matches everything)": {
			cs:       anyChar,
			expected: false,
		},
		"negated full range (matches nothing)": {
			cs:       charSet{Ranges: []runeRange{{Lo: 0, Hi: maxRune}}, Negated: true},
			expected: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cs.IsEmpty())
		})
	}
}

func TestCharSetIntersect(t *testing.T) {
	tests := map[string]struct {
		a, b     charSet
		expected charSet
	}{
		// both positive
		"two positive overlapping": {
			a:        normalized(false, runeRange{Lo: 'a', Hi: 'm'}),
			b:        normalized(false, runeRange{Lo: 'k', Hi: 'z'}),
			expected: charSet{Ranges: []runeRange{{Lo: 'k', Hi: 'm'}}},
		},
		"two positive disjoint": {
			a:        normalized(false, runeRange{Lo: 'a', Hi: 'c'}),
			b:        normalized(false, runeRange{Lo: 'x', Hi: 'z'}),
			expected: charSet{Ranges: nil},
		},
		// !A and B
		"negated and positive": {
			// !{/} and {a-z} = {a-z} (since / is not in a-z)
			a:        anyNonSlash,
			b:        normalized(false, runeRange{Lo: 'a', Hi: 'z'}),
			expected: charSet{Ranges: []runeRange{{Lo: 'a', Hi: 'z'}}},
		},
		"negated slash and positive slash": {
			// !{/} and {/} = empty
			a:        anyNonSlash,
			b:        singleChar('/'),
			expected: charSet{Ranges: nil},
		},
		// A and !B
		"positive and negated": {
			// {a-z} and !{m-p} = {a-l, q-z}
			a: normalized(false, runeRange{Lo: 'a', Hi: 'z'}),
			b: normalized(true, runeRange{Lo: 'm', Hi: 'p'}),
			expected: charSet{Ranges: []runeRange{
				{Lo: 'a', Hi: 'l'},
				{Lo: 'q', Hi: 'z'},
			}},
		},
		// both negated
		"two negated": {
			// !{a} and !{b} = !{a,b}
			a: normalized(true, runeRange{Lo: 'a', Hi: 'a'}),
			b: normalized(true, runeRange{Lo: 'b', Hi: 'b'}),
			expected: charSet{
				Ranges:  []runeRange{{Lo: 'a', Hi: 'b'}},
				Negated: true,
			},
		},
		"any intersect any non-slash": {
			a: anyChar,
			b: anyNonSlash,
			expected: charSet{
				Ranges:  []runeRange{{Lo: '/', Hi: '/'}},
				Negated: true,
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.a.Intersect(tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSubtractRanges(t *testing.T) {
	tests := map[string]struct {
		a, b     []runeRange
		expected []runeRange
	}{
		"no subtraction": {
			a:        []runeRange{{Lo: 'a', Hi: 'z'}},
			b:        nil,
			expected: []runeRange{{Lo: 'a', Hi: 'z'}},
		},
		"subtract beyond range": {
			a:        []runeRange{{Lo: 'a', Hi: 'c'}},
			b:        []runeRange{{Lo: 'x', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'c'}},
		},
		"subtract beginning": {
			a:        []runeRange{{Lo: 'a', Hi: 'z'}},
			b:        []runeRange{{Lo: 'a', Hi: 'c'}},
			expected: []runeRange{{Lo: 'd', Hi: 'z'}},
		},
		"subtract middle": {
			a:        []runeRange{{Lo: 'a', Hi: 'z'}},
			b:        []runeRange{{Lo: 'm', Hi: 'p'}},
			expected: []runeRange{{Lo: 'a', Hi: 'l'}, {Lo: 'q', Hi: 'z'}},
		},
		"subtract end": {
			a:        []runeRange{{Lo: 'a', Hi: 'z'}},
			b:        []runeRange{{Lo: 'x', Hi: 'z'}},
			expected: []runeRange{{Lo: 'a', Hi: 'w'}},
		},
		"subtract all": {
			a:        []runeRange{{Lo: 'a', Hi: 'z'}},
			b:        []runeRange{{Lo: 'a', Hi: 'z'}},
			expected: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := subtractRanges(tc.a, tc.b)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAnyNonSlashExcludesSlash(t *testing.T) {
	cs := anyNonSlash
	// Intersect with just '/' should be empty
	slashOnly := singleChar('/')
	result := cs.Intersect(slashOnly)
	assert.True(t, result.IsEmpty())

	// Intersect with 'a' should be non-empty
	aOnly := singleChar('a')
	result = cs.Intersect(aOnly)
	assert.False(t, result.IsEmpty())
}
