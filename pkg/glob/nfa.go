package glob

// NFAs are graphs of states connected by transitions. A transition either
// matches a character (charSet) or is an epsilon — a free move with no input
// consumed. The pattern "ab" looks like:
//
//	[s0] --'a'--> [s1] --'b'--> [s2*]     (* = accepting)
//
// To check if a string matches, track the set of active states and advance
// them one character at a time. Accept if any active state is accepting at
// the end.
//
// Epsilons let the NFA branch without consuming input — e.g. the two branches
// of {a,b} are each reached via an epsilon from the entry state (see build.go).
// The epsilon closure of a state is all states reachable from it for free:
//
//	[s0] --ε--> [s1] --'a'--> [s3*]
//	     --ε--> [s2*]
//
// Here the closure of s0 is {s0, s1, s2}, so this NFA accepts the empty string.
// Epsilons are eliminated before intersection testing (see eliminateEpsilon).

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

// epsilonClosure computes the set of states reachable from the given
// states via epsilon transitions only.
func epsilonClosure(states ...*state) []*state {
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
		closures[s.id] = epsilonClosure(s)
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
	stack := []*state{n.start}
	for len(stack) > 0 {
		s := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[s.id] {
			continue
		}
		seen[s.id] = true
		result = append(result, s)
		for _, t := range s.trans {
			stack = append(stack, t.to)
		}
	}
	return result
}
