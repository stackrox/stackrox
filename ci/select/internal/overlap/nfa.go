package overlap

import (
	"regexp/syntax"
	"slices"
	"unicode"
)

// rangeTransition is one labeled edge of the state machine: any character
// from low through high leads to nextState.
type rangeTransition struct {
	low, high rune
	nextState int
}

// nfaState is one node of an nfa. A node has two kinds of outgoing edges:
// rangeTransitions consume one character; epsilonEdges jump to another
// node without consuming anything, which is how alternation and repetition
// are wired without duplicating subgraphs.
type nfaState struct {
	epsilonEdges     []int
	rangeTransitions []rangeTransition
}

// nfa is a regular expression compiled into a graph. Each rule's inclusion
// and exclusion patterns each become one nfa; we use them to answer
// "can this set of patterns match a file the other set also matches" by
// graph algorithms instead of guessing at strings. start is the entry
// node, accept the node that means the whole path matched. conservative
// marks an nfa that over-approximates its pattern (see the package comment).
type nfa struct {
	states        []nfaState
	start, accept int
	conservative  bool
}

// nfaBuilder accumulates the nfa while the pattern tree is walked.
type nfaBuilder struct{ nfa *nfa }

// nfaFragment is the entry and exit node of a partially built subgraph,
// so composite patterns can connect fragments without knowing their size.
type nfaFragment struct{ start, accept int }

// newState appends a fresh isolated node and returns its index.
func (b *nfaBuilder) newState() int {
	b.nfa.states = append(b.nfa.states, nfaState{})
	return len(b.nfa.states) - 1
}

// addEpsilon connects two nodes with a free jump that consumes no character.
func (b *nfaBuilder) addEpsilon(from, to int) {
	b.nfa.states[from].epsilonEdges = append(b.nfa.states[from].epsilonEdges, to)
}

// addRanges adds one edge per low/high pair in pairs, each leading to to.
func (b *nfaBuilder) addRanges(from int, pairs []rune, to int) {
	for i := 0; i+1 < len(pairs); i += 2 {
		b.nfa.states[from].rangeTransitions = append(b.nfa.states[from].rangeTransitions, rangeTransition{low: pairs[i], high: pairs[i+1], nextState: to})
	}
}

// anyString matches every string and marks the NFA conservative; it is the
// fallback for constructs the walker cannot model exactly.
func (b *nfaBuilder) anyString() nfaFragment {
	s, a := b.newState(), b.newState()
	b.addRanges(s, []rune{0, unicode.MaxRune}, s)
	b.addEpsilon(s, a)
	b.nfa.conservative = true
	return nfaFragment{s, a}
}

var anyRunes = []rune{0, unicode.MaxRune}

// walk compiles one syntax tree into the NFA under construction.
func (b *nfaBuilder) walk(re *syntax.Regexp) nfaFragment {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpBeginLine:
		// The automaton starts at the beginning of the path, so a leading
		// '^' adds nothing. A later '^' is not modeled; compileUnion marks
		// that NFA conservative instead of treating the anchor as empty.
		s, a := b.newState(), b.newState()
		b.addEpsilon(s, a)
		return nfaFragment{s, a}
	case syntax.OpLiteral:
		s, a := b.newState(), b.newState()
		cur := s
		for _, r := range re.Rune {
			next := b.newState()
			if re.Flags&syntax.FoldCase != 0 {
				b.addRanges(cur, foldPairs(r), next)
			} else {
				b.addRanges(cur, []rune{r, r}, next)
			}
			cur = next
		}
		b.addEpsilon(cur, a)
		return nfaFragment{s, a}
	case syntax.OpCharClass:
		s, a := b.newState(), b.newState()
		b.addRanges(s, re.Rune, a)
		return nfaFragment{s, a}
	case syntax.OpAnyChar: // '.' under (?s)
		return b.simpleRanges(anyRunes)
	case syntax.OpAnyCharNotNL: // plain '.'
		return b.simpleRanges([]rune{0, '\n' - 1, '\n' + 1, unicode.MaxRune})
	case syntax.OpConcat:
		f := b.walk(re.Sub[0])
		for _, sub := range re.Sub[1:] {
			g := b.walk(sub)
			b.addEpsilon(f.accept, g.start)
			f = nfaFragment{f.start, g.accept}
		}
		return f
	case syntax.OpAlternate:
		s, a := b.newState(), b.newState()
		for _, sub := range re.Sub {
			f := b.walk(sub)
			b.addEpsilon(s, f.start)
			b.addEpsilon(f.accept, a)
		}
		return nfaFragment{s, a}
	case syntax.OpStar:
		f := b.walk(re.Sub[0])
		s, a := b.newState(), b.newState()
		b.addEpsilon(s, f.start)
		b.addEpsilon(s, a)
		b.addEpsilon(f.accept, f.start)
		b.addEpsilon(f.accept, a)
		return nfaFragment{s, a}
	case syntax.OpPlus:
		f := b.walk(re.Sub[0])
		b.addEpsilon(f.accept, f.start)
		return f
	case syntax.OpQuest:
		f := b.walk(re.Sub[0])
		s, a := b.newState(), b.newState()
		b.addEpsilon(s, f.start)
		b.addEpsilon(s, a)
		b.addEpsilon(f.accept, a)
		return nfaFragment{s, a}
	case syntax.OpCapture:
		return b.walk(re.Sub[0])
	default:
		// Counted repeats, word boundaries, and an end anchor the rewriter
		// left in place are not modeled. Over-approximate rather than
		// treat the gap as a proof.
		return b.anyString()
	}
}

func (b *nfaBuilder) simpleRanges(pairs []rune) nfaFragment {
	s, a := b.newState(), b.newState()
	b.addRanges(s, pairs, a)
	return nfaFragment{s, a}
}

// foldPairs returns singleton ranges covering r and its case variants.
func foldPairs(r rune) []rune {
	variants := map[rune]bool{r: true}
	for f := unicode.SimpleFold(r); f != r; f = unicode.SimpleFold(f) {
		variants[f] = true
	}
	pairs := make([]rune, 0, 2*len(variants))
	for v := range variants {
		pairs = append(pairs, v, v)
	}
	return pairs
}

// prefixToFullMatch turns a match-at-index-0 pattern into one the NFA can
// full-match. ok is false for a multiline '$', or a '$' that is not the
// end of its alternative. The caller then over-approximates.
func prefixToFullMatch(re *syntax.Regexp) (*syntax.Regexp, bool) {
	if containsOp(re, syntax.OpEndLine) {
		return nil, false
	}
	return prefixToFull(re)
}

// prefixToFull adds the free suffix inside each alternative. The suffix
// and a '$' apply only to the alternative that contains them.
func prefixToFull(re *syntax.Regexp) (*syntax.Regexp, bool) {
	switch re.Op {
	case syntax.OpAlternate:
		subs := make([]*syntax.Regexp, len(re.Sub))
		for i, sub := range re.Sub {
			next, ok := prefixToFull(sub)
			if !ok {
				return nil, false
			}
			subs[i] = next
		}
		return &syntax.Regexp{Op: syntax.OpAlternate, Flags: re.Flags, Sub: subs}, true
	case syntax.OpCapture:
		return prefixToFull(re.Sub[0])
	case syntax.OpConcat:
		return prefixConcat(re)
	case syntax.OpEndText:
		return &syntax.Regexp{Op: syntax.OpEmptyMatch, Flags: re.Flags}, true
	case syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		if containsOp(re, syntax.OpEndText) {
			return nil, false
		}
		return withFreeSuffix(re), true
	default:
		return withFreeSuffix(re), true
	}
}

// prefixConcat full-matches a concatenation. A trailing '$' ends the whole
// expression. A '$' nested in the last piece is rewritten there. A '$'
// earlier than that is not a single anchor, so the rewrite refuses it.
func prefixConcat(re *syntax.Regexp) (*syntax.Regexp, bool) {
	n := len(re.Sub)
	if n == 0 {
		return withFreeSuffix(re), true
	}
	last := re.Sub[n-1]
	head := slices.Clone(re.Sub[:n-1])
	if last.Op == syntax.OpEndText {
		if containsOp(joinConcat(head, re.Flags), syntax.OpEndText) {
			return nil, false
		}
		return joinConcat(head, re.Flags), true
	}
	if containsOp(last, syntax.OpEndText) {
		if containsOp(joinConcat(head, re.Flags), syntax.OpEndText) {
			return nil, false
		}
		rewritten, ok := prefixToFull(last)
		if !ok {
			return nil, false
		}
		return joinConcat(append(head, rewritten), re.Flags), true
	}
	if containsOp(re, syntax.OpEndText) {
		return nil, false
	}
	return withFreeSuffix(re), true
}

// withFreeSuffix matches re and then any remainder of the path, including
// a newline. The remainder is not part of the pattern, so it is not limited
// by the pattern's own '.'.
func withFreeSuffix(re *syntax.Regexp) *syntax.Regexp {
	anyChar := &syntax.Regexp{Op: syntax.OpAnyChar, Flags: syntax.DotNL}
	star := &syntax.Regexp{Op: syntax.OpStar, Flags: syntax.DotNL, Sub: []*syntax.Regexp{anyChar}}
	return &syntax.Regexp{Op: syntax.OpConcat, Flags: re.Flags, Sub: []*syntax.Regexp{re, star}}
}

func joinConcat(subs []*syntax.Regexp, flags syntax.Flags) *syntax.Regexp {
	switch len(subs) {
	case 0:
		return &syntax.Regexp{Op: syntax.OpEmptyMatch, Flags: flags}
	case 1:
		return subs[0]
	default:
		return &syntax.Regexp{Op: syntax.OpConcat, Flags: flags, Sub: subs}
	}
}

func containsOp(re *syntax.Regexp, op syntax.Op) bool {
	if re.Op == op {
		return true
	}
	return slices.ContainsFunc(re.Sub, func(sub *syntax.Regexp) bool {
		return containsOp(sub, op)
	})
}

// midBegin reports whether re has a '^' after the match has started.
// A leading '^' adds nothing: matching already begins at the start of the path.
func midBegin(re *syntax.Regexp, atStart bool) bool {
	switch re.Op {
	case syntax.OpBeginText, syntax.OpBeginLine:
		return !atStart
	case syntax.OpConcat:
		for _, sub := range re.Sub {
			if midBegin(sub, atStart) {
				return true
			}
			if canConsume(sub) {
				atStart = false
			}
		}
		return false
	case syntax.OpCapture, syntax.OpQuest:
		return midBegin(re.Sub[0], atStart)
	case syntax.OpAlternate:
		for _, sub := range re.Sub {
			if midBegin(sub, atStart) {
				return true
			}
		}
		return false
	case syntax.OpStar, syntax.OpPlus, syntax.OpRepeat:
		// A repeat can run again after the first iteration, so an anchor
		// in its body is no longer at the start.
		return containsBegin(re.Sub[0])
	default:
		return false
	}
}

func containsBegin(re *syntax.Regexp) bool {
	if re.Op == syntax.OpBeginText || re.Op == syntax.OpBeginLine {
		return true
	}
	return slices.ContainsFunc(re.Sub, containsBegin)
}

// canConsume reports whether re can match at least one character. Empty-width
// ops leave a following '^' at the start of the match.
func canConsume(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpEmptyMatch, syntax.OpBeginText, syntax.OpBeginLine,
		syntax.OpEndText, syntax.OpEndLine, syntax.OpWordBoundary, syntax.OpNoWordBoundary,
		syntax.OpNoMatch:
		return false
	case syntax.OpConcat, syntax.OpAlternate:
		return slices.ContainsFunc(re.Sub, canConsume)
	case syntax.OpCapture, syntax.OpStar, syntax.OpPlus, syntax.OpQuest, syntax.OpRepeat:
		return canConsume(re.Sub[0])
	default:
		return true
	}
}

// compileUnion builds one NFA matching the union of the rewritten patterns.
func compileUnion(patterns []string) *nfa {
	b := &nfaBuilder{nfa: &nfa{}}
	s, a := b.newState(), b.newState()
	for _, pattern := range patterns {
		f := b.compilePattern(pattern)
		b.addEpsilon(s, f.start)
		b.addEpsilon(f.accept, a)
	}
	b.nfa.start, b.nfa.accept = s, a
	return b.nfa
}

// compilePattern adds one pattern to the union being built. A pattern the
// rewriter or the walker cannot model becomes anyString, which matches
// every path and is not a proof.
func (b *nfaBuilder) compilePattern(pattern string) nfaFragment {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		// Unreachable: config validation compiles every pattern with the
		// production matcher before any set is built.
		return b.anyString()
	}
	full, ok := prefixToFullMatch(parsed.Simplify())
	if !ok {
		return b.anyString()
	}
	// A later '^' is compiled as an empty jump, which accepts paths the
	// real anchor rejects. Flag the NFA so callers over-approximate.
	if midBegin(full, true) {
		b.nfa.conservative = true
	}
	return b.walk(full)
}

// epsClosure appends epsilon-reachable states to set using seen as scratch.
// The returned slice aliases set; callers must copy closures they retain.
func epsClosure(n *nfa, set []int, seen []bool) []int {
	clear(seen)
	for _, s := range set {
		seen[s] = true
	}
	for i := 0; i < len(set); i++ {
		s := set[i]
		for _, to := range n.states[s].epsilonEdges {
			if !seen[to] {
				seen[to] = true
				set = append(set, to)
			}
		}
	}
	return set
}
