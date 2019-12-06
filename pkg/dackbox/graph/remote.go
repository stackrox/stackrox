package graph

import (
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
)

// RemoteReadable represents a shared graph state somewhere else that should be considered the current state of the
// RemoteGraph object.
type RemoteReadable interface {
	Read(reader func(graph RGraph))
}

// NewRemoteGraph returns an instance of a RemoteGraph with the input RemoteReadable object considered the
// unmodified state.
func NewRemoteGraph(reader RemoteReadable) *RemoteGraph {
	return &RemoteGraph{
		modified:     NewModifiedGraph(),
		remoteToRead: reader,
	}
}

// RemoteGraph is a representation of modifications made on top of a read-only graph.
type RemoteGraph struct {
	modified *ModifiedGraph

	readForward  sortedkeys.SortedKeys
	readBackward sortedkeys.SortedKeys

	remoteToRead RemoteReadable
}

// HasRefsFrom returns if there is an entry with 0 or more child keys in the graph.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) HasRefsFrom(from []byte) bool {
	rm.ensureFrom(from)
	return rm.modified.HasRefsFrom(from)
}

// HasRefsTo returns if there is an entry with 0 or more parent keys in the graph.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) HasRefsTo(to []byte) bool {
	rm.ensureTo(to)
	return rm.modified.HasRefsTo(to)
}

// CountRefsFrom returns the number of children reference from the input parent key.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) CountRefsFrom(from []byte) int {
	rm.ensureFrom(from)
	return rm.modified.CountRefsFrom(from)
}

// CountRefsTo returns the number of parents that reference the input child key.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) CountRefsTo(to []byte) int {
	rm.ensureTo(to)
	return rm.modified.CountRefsTo(to)
}

// GetRefsFrom returns the children referenced by the input parent key.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) GetRefsFrom(from []byte) [][]byte {
	rm.ensureFrom(from)
	return rm.modified.GetRefsFrom(from)
}

// GetRefsTo returns the parents that reference the input child key.
// Will cause a read to the remote Graph if this input key has not already been read, and update the underlying graph.
func (rm *RemoteGraph) GetRefsTo(to []byte) [][]byte {
	rm.ensureTo(to)
	return rm.modified.GetRefsTo(to)
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
// All keys affected by the change (parent and existing and new children) will have their states read if not already read
// so that the modification is consistent with the current remote state.
func (rm *RemoteGraph) SetRefs(from []byte, to [][]byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, to)
	rm.modified.SetRefs(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
// The remote state for all input keys will be read if not already read to ensure a consistent update.
func (rm *RemoteGraph) AddRefs(from []byte, to ...[]byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, to)
	rm.modified.AddRefs(from, to...)
}

// DeleteRefs removes all children from the input key, and removes the input key from the maps.
// The remote state will be read for the input key to make sure the modification is consistent.
func (rm *RemoteGraph) DeleteRefs(from []byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, nil)
	rm.modified.DeleteRefs(from)
}

// Clear removes all local edits to the remote state.
func (rm *RemoteGraph) Clear() {
	rm.modified.Clear()
}

func (rm *RemoteGraph) copyNeededState(from []byte, to [][]byte) { // Copy the current from value from the underlying state.
	rm.ensureFrom(from)
	rm.ensureToAll(rm.GetRefsFrom(from))
	rm.ensureToAll(to)
}

func (rm *RemoteGraph) ensureFrom(from []byte) {
	var inserted bool
	rm.readForward, inserted = rm.readForward.Insert(from)
	if !inserted {
		return
	}
	rm.remoteToRead.Read(func(g RGraph) {
		if g.HasRefsFrom(from) {
			rm.modified.underlying.initializeFrom(from, g.GetRefsFrom(from))
		}
	})
}

func (rm *RemoteGraph) ensureTo(to []byte) {
	var inserted bool
	rm.readBackward, inserted = rm.readBackward.Insert(to)
	if !inserted {
		return
	}
	rm.remoteToRead.Read(func(g RGraph) {
		if g.HasRefsTo(to) {
			rm.modified.underlying.initializeTo(to, g.GetRefsTo(to))
		}
	})
}

func (rm *RemoteGraph) ensureToAll(tos [][]byte) {
	if len(tos) == 0 {
		return
	}

	sortedTos := sortedkeys.Sort(tos)
	unfetched := sortedTos.Difference(rm.readBackward)
	rm.readBackward = rm.readBackward.Union(unfetched)

	// Read them all at once.
	rm.remoteToRead.Read(func(g RGraph) {
		for _, to := range unfetched {
			rm.readBackward, _ = rm.readBackward.Insert(to)
			if g.HasRefsTo(to) {
				rm.modified.underlying.initializeTo(to, g.GetRefsTo(to))
			}
		}
	})
}
