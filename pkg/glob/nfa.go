package glob

// state represents a state in an NFA.
type state struct {
	id        int
	accepting bool
	trans     []transition
}

// transition represents a transition from one state to another.
// Either epsilon is true (unconditional transition) or chars defines
// the set of characters that trigger the transition.
type transition struct {
	chars   charSet
	epsilon bool
	to      *state
}

// nfa represents a non-deterministic finite automaton with a single
// start state and a single accept state (Thompson's construction guarantee).
type nfa struct {
	start  *state
	accept *state
}

// stateAllocator produces states with unique IDs.
type stateAllocator struct {
	nextID int
}

func (a *stateAllocator) newState() *state {
	s := &state{id: a.nextID}
	a.nextID++
	return s
}

func (a *stateAllocator) newAcceptState() *state {
	s := a.newState()
	s.accepting = true
	return s
}

// accepts reports whether the NFA accepts the given string.
// Used for testing.
func (n *nfa) accepts(s string) bool {
	current := epsilonClosure([]*state{n.start})
	for _, r := range s {
		var next []*state
		for _, st := range current {
			for _, t := range st.trans {
				if t.epsilon {
					continue
				}
				if charSetContains(t.chars, r) {
					next = append(next, t.to)
				}
			}
		}
		if len(next) == 0 {
			return false
		}
		current = epsilonClosure(next)
	}
	for _, st := range current {
		if st.accepting {
			return true
		}
	}
	return false
}

// charSetContains reports whether the charSet contains the given rune.
func charSetContains(cs charSet, r rune) bool {
	found := false
	for _, rr := range cs.Ranges {
		if r >= rr.Lo && r <= rr.Hi {
			found = true
			break
		}
	}
	if cs.Negated {
		return !found
	}
	return found
}

// epsilonClosure computes the set of states reachable from the given
// states via epsilon transitions only.
func epsilonClosure(states []*state) []*state {
	seen := make(map[int]bool)
	var result []*state
	stack := append([]*state(nil), states...)
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[s.id] {
			continue
		}
		seen[s.id] = true
		result = append(result, s)
		for _, t := range s.trans {
			if t.epsilon && !seen[t.to.id] {
				stack = append(stack, t.to)
			}
		}
	}
	return result
}

// eliminateEpsilon returns a new NFA with no epsilon transitions that
// accepts the same language as the input NFA.
func eliminateEpsilon(n *nfa) *nfa {
	// Collect all states from the original NFA.
	allStates := collectStates(n)

	// Compute epsilon closures for each state.
	closures := make(map[int][]*state)
	for _, s := range allStates {
		closures[s.id] = epsilonClosure([]*state{s})
	}

	// Build new states, preserving original IDs.
	newStates := make(map[int]*state)
	for _, s := range allStates {
		ns := &state{id: s.id}
		for _, cs := range closures[s.id] {
			if cs.accepting {
				ns.accepting = true
				break
			}
		}
		newStates[s.id] = ns
	}

	// Build transitions: for each state, look at the epsilon closure,
	// then follow non-epsilon transitions from those states.
	for _, s := range allStates {
		ns := newStates[s.id]
		for _, closureState := range closures[s.id] {
			for _, t := range closureState.trans {
				if t.epsilon {
					continue
				}
				ns.trans = append(ns.trans, transition{
					chars: t.chars,
					to:    newStates[t.to.id],
				})
			}
		}
	}

	newStart := newStates[n.start.id]
	newAccept := newStates[n.accept.id]

	return &nfa{start: newStart, accept: newAccept}
}

// collectStates returns all states reachable from the NFA's start state.
func collectStates(n *nfa) []*state {
	seen := make(map[int]bool)
	var result []*state
	var walk func(s *state)
	walk = func(s *state) {
		if seen[s.id] {
			return
		}
		seen[s.id] = true
		result = append(result, s)
		for _, t := range s.trans {
			walk(t.to)
		}
	}
	walk(n.start)
	return result
}
