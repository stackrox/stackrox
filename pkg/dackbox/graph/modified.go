package graph

import (
	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
)

// Modification represents a readable change to a Graph.
type Modification interface {
	RGraph

	FromModified([]byte) bool
	ToModified([]byte) bool

	Apply(graph applyableGraph)
}

// NewModifiedGraph returns a new instance of a ModifiedGraph.
func NewModifiedGraph(graph RWGraph) *ModifiedGraph {
	return &ModifiedGraph{
		RWGraph: graph,
	}
}

// ModifiedGraph provides a view of an RWGraph that tracks changes.
// Anytime a key is modified in the graph, the affected values are recorded, and the changes can be played back onto
// an 'applyableGraph' with the Apply function.
type ModifiedGraph struct {
	RWGraph

	modifiedFrom sortedkeys.SortedKeys
	modifiedTo   sortedkeys.SortedKeys
}

// Apply applies a modification to a separate graph object.
func (ms *ModifiedGraph) Apply(graph applyableGraph) {
	for _, from := range ms.modifiedFrom {
		tos := ms.GetRefsFrom(from)
		if tos == nil {
			graph.deleteFrom(from)
		} else {
			graph.setFrom(from, tos)
		}
	}
	for _, to := range ms.modifiedTo {
		froms := ms.GetRefsTo(to)
		if froms == nil {
			graph.deleteTo(to)
		} else {
			graph.setTo(to, froms)
		}
	}
}

// FromModified returns of the children of the input key have been modified in the ModifiedGraph.
func (ms *ModifiedGraph) FromModified(from []byte) bool {
	return ms.modifiedFrom.Find(from) != -1
}

// ToModified returns of the parents of the input key have been modified in the ModifiedGraph.
func (ms *ModifiedGraph) ToModified(to []byte) bool {
	return ms.modifiedTo.Find(to) != -1
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
// Will add all of the input keys, we well as any keys that were previously children of 'from' to the list of values modified.
func (ms *ModifiedGraph) SetRefs(from []byte, to [][]byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(sortedkeys.Sort(to))
	ms.modifiedTo = ms.modifiedTo.Union(ms.GetRefsFrom(from))

	ms.RWGraph.SetRefs(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
// Will add all of the input keys to the list of values modified.
func (ms *ModifiedGraph) AddRefs(from []byte, to ...[]byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(sortedkeys.Sort(to))

	ms.RWGraph.AddRefs(from, to...)
}

// DeleteRefsFrom removes all children from the input key, and removes the input key from the maps.
// The key and it's current list of children will be added to the lists of modified values.
func (ms *ModifiedGraph) DeleteRefsFrom(from []byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(ms.GetRefsFrom(from))

	ms.RWGraph.DeleteRefsFrom(from)
}

// DeleteRefsTo removes all parents from the input key, and removes the input key from the maps.
// The key and it's current list of children will be added to the lists of modified values.
func (ms *ModifiedGraph) DeleteRefsTo(to []byte) {
	ms.modifiedTo, _ = ms.modifiedTo.Insert(to)
	ms.modifiedFrom = ms.modifiedFrom.Union(ms.GetRefsTo(to))

	ms.RWGraph.DeleteRefsTo(to)
}
