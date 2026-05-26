package glob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// nfaAccepts reports whether the NFA accepts the given string.
func nfaAccepts(n *nfa, s string) bool {
	current := epsilonClosure(n.start)
	for _, r := range s {
		var next []*state
		for _, st := range current {
			for _, t := range st.trans {
				if t.epsilon {
					continue
				}
				if t.chars.contains(r) {
					next = append(next, t.to)
				}
			}
		}
		if len(next) == 0 {
			return false
		}
		current = epsilonClosure(next...)
	}
	for _, st := range current {
		if st.accepting {
			return true
		}
	}
	return false
}

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
		transition{chars: anyNonSlash, to: start},
	)
	return &nfa{start: start, accept: accept}
}

func TestNFAAcceptsLiteral(t *testing.T) {
	n := buildLiteralNFA("abc")
	assert.True(t, nfaAccepts(n, "abc"))
	assert.False(t, nfaAccepts(n, "ab"))
	assert.False(t, nfaAccepts(n, "abcd"))
	assert.False(t, nfaAccepts(n, ""))
	assert.False(t, nfaAccepts(n, "xyz"))
}

func TestNFAAcceptsStar(t *testing.T) {
	n := buildStarNFA()
	assert.True(t, nfaAccepts(n, ""))
	assert.True(t, nfaAccepts(n, "a"))
	assert.True(t, nfaAccepts(n, "abc"))
	assert.False(t, nfaAccepts(n, "a/b"))
	assert.False(t, nfaAccepts(n, "/"))
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
	assert.True(t, nfaAccepts(n, "a"))
	assert.False(t, nfaAccepts(n, "b"))
	assert.False(t, nfaAccepts(n, ""))
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
	assert.True(t, nfaAccepts(original, ""))
	assert.True(t, nfaAccepts(original, "a"))
	assert.False(t, nfaAccepts(original, "b"))
	assert.False(t, nfaAccepts(original, "aa"))

	// Eliminate epsilons
	eliminated := eliminateEpsilon(original)

	// Verify no epsilon transitions remain
	for _, s := range collectStates(eliminated) {
		for _, tr := range s.trans {
			assert.False(t, tr.epsilon, "found epsilon transition in eliminated NFA")
		}
	}

	// Verify same language
	assert.True(t, nfaAccepts(eliminated, ""))
	assert.True(t, nfaAccepts(eliminated, "a"))
	assert.False(t, nfaAccepts(eliminated, "b"))
	assert.False(t, nfaAccepts(eliminated, "aa"))
}

func TestEliminateEpsilonStar(t *testing.T) {
	star := buildStarNFA()
	eliminated := eliminateEpsilon(star)

	// Same language
	assert.True(t, nfaAccepts(eliminated, ""))
	assert.True(t, nfaAccepts(eliminated, "a"))
	assert.True(t, nfaAccepts(eliminated, "abc"))
	assert.False(t, nfaAccepts(eliminated, "a/b"))
	assert.False(t, nfaAccepts(eliminated, "/"))
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
			cs: anyChar, r: '/', expected: true,
		},
		"any matches letter": {
			cs: anyChar, r: 'x', expected: true,
		},
		"any non-slash matches letter": {
			cs: anyNonSlash, r: 'x', expected: true,
		},
		"any non-slash rejects slash": {
			cs: anyNonSlash, r: '/', expected: false,
		},
		"range match": {
			cs:       normalized(false, runeRange{Lo: 'a', Hi: 'z'}),
			r:        'm',
			expected: true,
		},
		"range no match": {
			cs:       normalized(false, runeRange{Lo: 'a', Hi: 'z'}),
			r:        'A',
			expected: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cs.contains(tc.r))
		})
	}
}
