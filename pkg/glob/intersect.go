package glob

// statePair identifies a pair of states from two NFAs.
type statePair struct {
	a, b int
}

const maxStatePairs = 10_000

// intersectionNonEmpty reports whether the intersection of two
// epsilon-free NFAs accepts any string. Uses lazy product construction
// with breadth first search — only explores reachable state pairs. Returns true
// (conservatively assumes overlap) if the state-pair limit is exceeded.
func intersectionNonEmpty(n1, n2 *nfa) bool {
	visited := make(map[statePair]bool)
	queue := []statePair{{n1.start.id, n2.start.id}}

	states1 := indexStates(n1)
	states2 := indexStates(n2)

	// Check if start pair is accepting.
	if states1[n1.start.id].accepting && states2[n2.start.id].accepting {
		return true
	}

	for len(queue) > 0 {
		pair := queue[0]
		queue = queue[1:]

		if visited[pair] {
			continue
		}
		if len(visited) >= maxStatePairs {
			return true
		}
		visited[pair] = true

		s1 := states1[pair.a]
		s2 := states2[pair.b]

		for _, t1 := range s1.trans {
			for _, t2 := range s2.trans {
				if t1.chars.Intersect(t2.chars).IsEmpty() {
					continue
				}

				next := statePair{t1.to.id, t2.to.id}
				if states1[next.a].accepting && states2[next.b].accepting {
					return true
				}
				if !visited[next] {
					queue = append(queue, next)
				}
			}
		}
	}

	return false
}

// indexStates builds a map from state ID to state for fast lookup.
func indexStates(n *nfa) map[int]*state {
	states := collectStates(n)
	index := make(map[int]*state, len(states))
	for _, s := range states {
		index[s.id] = s
	}
	return index
}
