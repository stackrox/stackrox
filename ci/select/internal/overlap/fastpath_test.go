package overlap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDisjointStarts(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		left, right string
		disjoint    bool
	}{
		"should prove the repeated-suffix rules disjoint": {"^[ab]*a[ab]{16}$", "^docs/", true},
		"should preserve a shared empty string":           {"^a?$", "^b?$", false},
		"should allow only one side to match empty":       {"^a?$", "^b$", true},
		"should preserve two empty-only languages":        {"^$", "^$", false},
		"should separate empty-only and nonempty words":   {"^$", "^b$", true},
		"should include transitions after nullable parts": {"^a*/", "^/b", false},
		"should include every alternative":                {"^(a|b)$", "^b$", false},
		"should include folded-case characters":           {"^(?i:a)$", "^A$", false},
		"should compare Unicode ranges":                   {"^é$", "^猫$", true},
		"should leave shared prefixes to full analysis":   {"^ab$", "^ac$", false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			left, right := compileUnion([]string{c.left}), compileUnion([]string{c.right})
			assert.Equal(t, c.disjoint, disjointStarts(left, right))
			assert.Equal(t, c.disjoint, disjointStarts(right, left))
		})
	}
}

func TestNFAHasMatch(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		patterns []string
		matches  bool
	}{
		"should find a repeated-suffix word": {[]string{"^[ab]*a[ab]{16}$"}, true},
		"should accept an empty string":      {[]string{"^$"}, true},
		"should find a word through a loop":  {[]string{"^a*b$"}, true},
		"should reject an empty union":       {nil, false},
		"should reject surrogate-only words": {[]string{`^\x{D800}$`}, false},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.matches, compileUnion(c.patterns).hasMatch())
		})
	}
	t.Run("should terminate on an empty epsilon cycle", func(t *testing.T) {
		t.Parallel()
		n := &nfa{states: []nfaState{{epsilonEdges: []int{0}}, {}}, start: 0, accept: 1}
		assert.False(t, n.hasMatch())
		n.states[0].epsilonEdges = append(n.states[0].epsilonEdges, 1)
		assert.True(t, n.hasMatch())
	})
}
