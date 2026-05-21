package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// buildLiteralNFA builds an NFA that matches the exact string s.
func buildLiteralNFA(s string) *nfa {
	alloc := &stateAllocator{}
	start := alloc.newState()
	current := start
	for _, r := range s {
		next := alloc.newState()
		current.trans = append(current.trans, transition{
			chars: singleChar(r),
			to:    next,
		})
		current = next
	}
	current.accepting = true
	return &nfa{start: start, accept: current}
}

// buildStarNFA builds an NFA matching "any non-slash" zero or more times (like glob *).
func buildStarNFA() *nfa {
	alloc := &stateAllocator{}
	start := alloc.newState()
	accept := alloc.newAcceptState()
	start.trans = append(start.trans,
		transition{epsilon: true, to: accept},
		transition{chars: anyNonSlash(), to: start},
	)
	return &nfa{start: start, accept: accept}
}

func TestNFAAcceptsLiteral(t *testing.T) {
	n := buildLiteralNFA("abc")
	assert.True(t, n.accepts("abc"))
	assert.False(t, n.accepts("ab"))
	assert.False(t, n.accepts("abcd"))
	assert.False(t, n.accepts(""))
	assert.False(t, n.accepts("xyz"))
}

func TestNFAAcceptsStar(t *testing.T) {
	n := buildStarNFA()
	assert.True(t, n.accepts(""))
	assert.True(t, n.accepts("a"))
	assert.True(t, n.accepts("abc"))
	assert.False(t, n.accepts("a/b"))
	assert.False(t, n.accepts("/"))
}

func TestNFAAcceptsWithEpsilon(t *testing.T) {
	// Build: epsilon -> "a" -> epsilon -> accept
	alloc := &stateAllocator{}
	s0 := alloc.newState()
	s1 := alloc.newState()
	s2 := alloc.newState()
	s3 := alloc.newAcceptState()

	s0.trans = append(s0.trans, transition{epsilon: true, to: s1})
	s1.trans = append(s1.trans, transition{chars: singleChar('a'), to: s2})
	s2.trans = append(s2.trans, transition{epsilon: true, to: s3})

	n := &nfa{start: s0, accept: s3}
	assert.True(t, n.accepts("a"))
	assert.False(t, n.accepts("b"))
	assert.False(t, n.accepts(""))
}

func TestEliminateEpsilon(t *testing.T) {
	// Build an NFA with epsilon transitions that accepts "a" or ""
	// start --ε--> s1
	// s1 --'a'--> s2
	// s1 --ε--> accept
	// s2 --ε--> accept
	alloc := &stateAllocator{}
	start := alloc.newState()
	s1 := alloc.newState()
	s2 := alloc.newState()
	accept := alloc.newAcceptState()

	start.trans = append(start.trans, transition{epsilon: true, to: s1})
	s1.trans = append(s1.trans,
		transition{chars: singleChar('a'), to: s2},
		transition{epsilon: true, to: accept},
	)
	s2.trans = append(s2.trans, transition{epsilon: true, to: accept})

	original := &nfa{start: start, accept: accept}

	// Verify original works
	assert.True(t, original.accepts(""))
	assert.True(t, original.accepts("a"))
	assert.False(t, original.accepts("b"))
	assert.False(t, original.accepts("aa"))

	// Eliminate epsilons
	eliminated := eliminateEpsilon(original)

	// Verify no epsilon transitions remain
	for _, s := range collectStates(eliminated) {
		for _, tr := range s.trans {
			assert.False(t, tr.epsilon, "found epsilon transition in eliminated NFA")
		}
	}

	// Verify same language
	assert.True(t, eliminated.accepts(""))
	assert.True(t, eliminated.accepts("a"))
	assert.False(t, eliminated.accepts("b"))
	assert.False(t, eliminated.accepts("aa"))
}

func TestEliminateEpsilonStar(t *testing.T) {
	star := buildStarNFA()
	eliminated := eliminateEpsilon(star)

	// Same language
	assert.True(t, eliminated.accepts(""))
	assert.True(t, eliminated.accepts("a"))
	assert.True(t, eliminated.accepts("abc"))
	assert.False(t, eliminated.accepts("a/b"))
	assert.False(t, eliminated.accepts("/"))
}

func TestCharSetContains(t *testing.T) {
	tests := map[string]struct {
		cs       charSet
		r        rune
		expected bool
	}{
		"single match": {
			cs: singleChar('a'), r: 'a', expected: true,
		},
		"single no match": {
			cs: singleChar('a'), r: 'b', expected: false,
		},
		"any matches slash": {
			cs: anyChar(), r: '/', expected: true,
		},
		"any matches letter": {
			cs: anyChar(), r: 'x', expected: true,
		},
		"any non-slash matches letter": {
			cs: anyNonSlash(), r: 'x', expected: true,
		},
		"any non-slash rejects slash": {
			cs: anyNonSlash(), r: '/', expected: false,
		},
		"range match": {
			cs:       fromRanges(false, runeRange{Lo: 'a', Hi: 'z'}),
			r:        'm',
			expected: true,
		},
		"range no match": {
			cs:       fromRanges(false, runeRange{Lo: 'a', Hi: 'z'}),
			r:        'A',
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, charSetContains(tc.cs, tc.r))
		})
	}
}
