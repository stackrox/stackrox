package graph

import (
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/utils"
)

// NewModifiedGraph returns a new instance of a ModifiedGraph.
func NewModifiedGraph() *ModifiedGraph {
	return &ModifiedGraph{
		underlying: NewGraph(),
	}
}

// ModifiedGraph provides a view of a Graph that tracks changes.
type ModifiedGraph struct {
	underlying *Graph

	modifiedFrom sortedkeys.SortedKeys
	modifiedTo   sortedkeys.SortedKeys
}

// HasRefsFrom returns if there is an entry with 0 or more child keys in the graph.
func (ms *ModifiedGraph) HasRefsFrom(from []byte) bool {
	return ms.underlying.HasRefsFrom(from)
}

// HasRefsTo returns if there is an entry with 0 or more parent keys in the graph.
func (ms *ModifiedGraph) HasRefsTo(to []byte) bool {
	return ms.underlying.HasRefsTo(to)
}

// CountRefsFrom returns the number of children reference from the input parent key.
func (ms *ModifiedGraph) CountRefsFrom(from []byte) int {
	return ms.underlying.CountRefsFrom(from)
}

// CountRefsTo returns the number of parents that reference the input child key.
func (ms *ModifiedGraph) CountRefsTo(to []byte) int {
	return ms.underlying.CountRefsTo(to)
}

// GetRefsFrom returns the children referenced by the input parent key.
func (ms *ModifiedGraph) GetRefsFrom(from []byte) [][]byte {
	return ms.underlying.GetRefsFrom(from)
}

// GetRefsTo returns the parents that reference the input child key.
func (ms *ModifiedGraph) GetRefsTo(to []byte) [][]byte {
	return ms.underlying.GetRefsTo(to)
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
// Will add all of the input keys, we well as any keys that were previously children of 'from' to the list of values modified.
func (ms *ModifiedGraph) SetRefs(from []byte, to [][]byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(sortedkeys.Sort(to))
	ms.modifiedTo = ms.modifiedTo.Union(ms.underlying.GetRefsFrom(from))

	ms.underlying.SetRefs(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
// Will add all of the input keys to the list of values modified.
func (ms *ModifiedGraph) AddRefs(from []byte, to ...[]byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(sortedkeys.Sort(to))

	ms.underlying.AddRefs(from, to...)
}

// DeleteRefs removes all children from the input key, and removes the input key from the maps.
// The key and it's current list of children will be added to the lists of modified values.
func (ms *ModifiedGraph) DeleteRefs(from []byte) {
	ms.modifiedFrom, _ = ms.modifiedFrom.Insert(from)
	ms.modifiedTo = ms.modifiedTo.Union(ms.underlying.GetRefsFrom(from))

	ms.underlying.DeleteRefs(from)
}

// Copy returns a new copy of the modified state.
func (ms *ModifiedGraph) Copy() *ModifiedGraph {
	return &ModifiedGraph{
		underlying:   ms.underlying.Copy(),
		modifiedFrom: utils.CopyKeys(ms.modifiedFrom),
		modifiedTo:   utils.CopyKeys(ms.modifiedTo),
	}
}

// Clear removes all values and modifications.
func (ms *ModifiedGraph) Clear() {
	ms.modifiedFrom = nil
	ms.modifiedTo = nil
	ms.underlying.Clear()
}
