package overlap

import "unicode/utf8"

// disjointStarts proves disjointness from initial transitions and empty-string
// acceptance. A shared first character leaves the answer to full analysis.
func disjointStarts(a, b *nfa) bool {
	left, leftEmpty := initialTransitions(a)
	right, rightEmpty := initialTransitions(b)
	if leftEmpty && rightEmpty {
		return false
	}
	for _, l := range left {
		for _, r := range right {
			if l.low <= r.high && r.low <= l.high {
				return false
			}
		}
	}
	return true
}

// initialTransitions includes transitions reachable without consuming a rune.
func initialTransitions(n *nfa) (transitions []rangeTransition, acceptsEmpty bool) {
	initial := epsClosure(n, []int{n.start}, make([]bool, len(n.states)))
	for _, state := range initial {
		acceptsEmpty = acceptsEmpty || state == n.accept
		transitions = append(transitions, n.states[state].rangeTransitions...)
	}
	return transitions, acceptsEmpty
}

// hasMatch reports whether n accepts a string, including the empty string.
// Surrogate-only transitions cannot consume a Go regexp input rune.
func (n *nfa) hasMatch() bool {
	seen := make([]bool, len(n.states))
	pending := []int{n.start}
	for len(pending) > 0 {
		state := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[state] {
			continue
		}
		seen[state] = true
		if state == n.accept {
			return true
		}
		pending = append(pending, n.states[state].epsilonEdges...)
		for _, edge := range n.states[state].rangeTransitions {
			if utf8.ValidRune(edge.low) || utf8.ValidRune(edge.high) {
				pending = append(pending, edge.nextState)
			}
		}
	}
	return false
}
