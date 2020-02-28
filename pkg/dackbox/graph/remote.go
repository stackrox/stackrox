package graph

import (
	"github.com/stackrox/rox/pkg/set"
)

// RemoteReadable represents a shared graph state somewhere else that should be considered the current state of the
// RemoteGraph object.
type RemoteReadable func(reader func(graph RGraph))

// NewRemoteGraph returns an instance of a RemoteGraph with the input RemoteReadable object considered the
// unlocal state.
func NewRemoteGraph(base RWGraph, reader RemoteReadable) *RemoteGraph {
	return &RemoteGraph{
		RWGraph:      base,
		readForward:  set.NewStringSet(),
		readBackward: set.NewStringSet(),
		remoteToRead: reader,
	}
}

// RemoteGraph is a representation of modifications made on top of a read-only graph.
type RemoteGraph struct {
	RWGraph

	readForward  set.StringSet
	readBackward set.StringSet

	remoteToRead RemoteReadable
}

// Underlying returns the base graph used to store read and modified data.
func (rm *RemoteGraph) Underlying() RWGraph {
	return rm.RWGraph
}

// HasRefsFrom returns if there is an entry with 0 or more child keys in the graph.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) HasRefsFrom(from []byte) bool {
	rm.ensureFrom(from)
	return rm.RWGraph.HasRefsFrom(from)
}

// HasRefsTo returns if there is an entry with 0 or more parent keys in the graph.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) HasRefsTo(to []byte) bool {
	rm.ensureTo(to)
	return rm.RWGraph.HasRefsTo(to)
}

// CountRefsFrom returns the number of children reference from the input parent key.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) CountRefsFrom(from []byte) int {
	rm.ensureFrom(from)
	return rm.RWGraph.CountRefsFrom(from)
}

// CountRefsTo returns the number of parents that reference the input child key.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) CountRefsTo(to []byte) int {
	rm.ensureTo(to)
	return rm.RWGraph.CountRefsTo(to)
}

// GetRefsFrom returns the children referenced by the input parent key.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) GetRefsFrom(from []byte) [][]byte {
	rm.ensureFrom(from)
	return rm.RWGraph.GetRefsFrom(from)
}

// GetRefsTo returns the parents that reference the input child key.
// Will cause a read to the remote Graph if this input key has not already been read.
func (rm *RemoteGraph) GetRefsTo(to []byte) [][]byte {
	rm.ensureTo(to)
	return rm.RWGraph.GetRefsTo(to)
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
// All keys affected by the change (parent and existing and new children) will have their states read if not already read
// so that the modification is consistent with the current remote state.
func (rm *RemoteGraph) SetRefs(from []byte, to [][]byte) error {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, to)
	return rm.RWGraph.SetRefs(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
// The remote state for all input keys will be read if not already read to ensure a consistent update.
func (rm *RemoteGraph) AddRefs(from []byte, to ...[]byte) error {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, to)
	return rm.RWGraph.AddRefs(from, to...)
}

// DeleteRefs removes all children from the input key, and removes the input key from the maps.
// The remote state will be read for the input key to make sure the modification is consistent.
func (rm *RemoteGraph) DeleteRefs(from []byte) error {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.copyNeededState(from, nil)
	return rm.RWGraph.DeleteRefs(from)
}

func (rm *RemoteGraph) copyNeededState(from []byte, to [][]byte) { // Copy the current from value from the underlying state.
	rm.ensureFrom(from)
	rm.ensureToAll(rm.GetRefsFrom(from))
	rm.ensureToAll(to)
}

func (rm *RemoteGraph) ensureFrom(from []byte) {
	strFrom := string(from)
	if !rm.readForward.Add(strFrom) {
		return
	}

	rm.remoteToRead(func(g RGraph) {
		if g.HasRefsFrom(from) {
			rm.RWGraph.setFrom(from, g.GetRefsFrom(from))
		}
	})
}

func (rm *RemoteGraph) ensureTo(to []byte) {
	strTo := string(to)
	if !rm.readBackward.Add(strTo) {
		return
	}

	rm.remoteToRead(func(g RGraph) {
		if g.HasRefsTo(to) {
			rm.RWGraph.setTo(to, g.GetRefsTo(to))
		}
	})
}

func (rm *RemoteGraph) ensureToAll(tos [][]byte) {
	if len(tos) == 0 {
		return
	}

	unfetched := make([][]byte, 0, len(tos))
	for _, to := range tos {
		strTo := string(to)
		if rm.readBackward.Add(strTo) {
			unfetched = append(unfetched, to)
		}
	}

	// Read them all at once.
	rm.remoteToRead(func(g RGraph) {
		for _, to := range unfetched {
			if g.HasRefsTo(to) {
				rm.RWGraph.setTo(to, g.GetRefsTo(to))
			}
		}
	})
}
