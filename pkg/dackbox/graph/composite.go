package graph

// NewCompositeGraph returns an ReadOnlyGraph instance that uses the input states as a stack to surface values.
// When a request for an id relationship is placed through one of it's functions, it searches down the stack for the
// first state that has the desired information, and returns that to the user.
func NewCompositeGraph(base RGraph, modifications ...Modification) RGraph {
	if len(modifications) == 0 {
		return base
	}
	return compositeGraph{
		modifications: modifications,
		baseState:     base,
	}
}

type compositeGraph struct {
	modifications []Modification
	baseState     RGraph
}

// HasRefsFrom returns if there is an entry with 0 or more child keys in the graph.
func (cs compositeGraph) HasRefsFrom(from []byte) bool {
	s := cs.getFirstStateWithFrom(from)
	return s.HasRefsFrom(from)
}

// HasRefsTo returns if there is an entry with 0 or more parent keys in the graph.
func (cs compositeGraph) HasRefsTo(to []byte) bool {
	s := cs.getFirstStateWithTo(to)
	return s.HasRefsTo(to)
}

// CountRefsFrom returns the number of children reference from the input parent key.
func (cs compositeGraph) CountRefsFrom(from []byte) int {
	s := cs.getFirstStateWithFrom(from)
	return s.CountRefsFrom(from)
}

// CountRefsTo returns the number of parents that reference the input child key.
func (cs compositeGraph) CountRefsTo(to []byte) int {
	s := cs.getFirstStateWithTo(to)
	return s.CountRefsTo(to)
}

// GetRefsFrom returns the children referenced by the input parent key.
func (cs compositeGraph) GetRefsFrom(from []byte) [][]byte {
	s := cs.getFirstStateWithFrom(from)
	return s.GetRefsFrom(from)
}

// GetRefsTo returns the parents that reference the input child key.
func (cs compositeGraph) GetRefsTo(to []byte) [][]byte {
	s := cs.getFirstStateWithTo(to)
	return s.GetRefsTo(to)
}

// getFirstStateWithFrom returns the graph with the most recent modification of the input key's children.
// Will always return a non-nil RGraph.
func (cs compositeGraph) getFirstStateWithFrom(from []byte) RGraph {
	match := cs.getFirstStateThatMatches(func(modded Modification) bool {
		return modded.FromModified(from)
	})
	if match == nil {
		return cs.baseState
	}
	return match
}

// getFirstStateWithTo returns the graph with the most recent modification of the input key's parents.
// Will always return a non-nil RGraph.
func (cs compositeGraph) getFirstStateWithTo(to []byte) RGraph {
	match := cs.getFirstStateThatMatches(func(modded Modification) bool {
		return modded.ToModified(to)
	})
	if match == nil {
		return cs.baseState
	}
	return match
}

func (cs compositeGraph) getFirstStateThatMatches(pred func(modded Modification) bool) RGraph {
	for i := len(cs.modifications) - 1; i >= 0; i-- {
		if pred(cs.modifications[i]) {
			return cs.modifications[i]
		}
	}
	return nil
}
