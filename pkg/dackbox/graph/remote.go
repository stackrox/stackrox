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

// GetRefsFromPrefix returns the children referenced by the input parent key filtered by prefix.
func (rm *RemoteGraph) GetRefsFromPrefix(from, prefix []byte) [][]byte {
	rm.ensureFrom(from)
	return rm.RWGraph.GetRefsFromPrefix(from, prefix)
}

// GetRefsToPrefix gets the references to the key filtered by prefix
func (rm *RemoteGraph) GetRefsToPrefix(to, prefix []byte) [][]byte {
	rm.ensureTo(to)
	return rm.RWGraph.GetRefsToPrefix(to, prefix)
}

// CountRefsFromPrefix returns the number of children referenced by the input parent key filtered by prefix.
func (rm *RemoteGraph) CountRefsFromPrefix(from, prefix []byte) int {
	rm.ensureFrom(from)
	return rm.RWGraph.CountRefsFromPrefix(from, prefix)
}

// CountRefsToPrefix gets the number of references to the key filtered by prefix
func (rm *RemoteGraph) CountRefsToPrefix(to, prefix []byte) int {
	rm.ensureTo(to)
	return rm.RWGraph.CountRefsToPrefix(to, prefix)
}

// ReferencedFromPrefix returns whether there exists a reference to this key with the specific prefix
func (rm *RemoteGraph) ReferencedFromPrefix(to, prefix []byte) bool {
	rm.ensureTo(to)
	return rm.RWGraph.ReferencedFromPrefix(to, prefix)
}

// ReferencesPrefix returns whether a reference from the current key
// to a key with the specified prefix exists in the graph.
func (rm *RemoteGraph) ReferencesPrefix(from, prefix []byte) bool {
	rm.ensureFrom(from)
	return rm.RWGraph.ReferencesPrefix(from, prefix)
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
// All keys affected by the change (parent and existing and new children) will have their states read if not already read
// so that the modification is consistent with the current remote state.
func (rm *RemoteGraph) SetRefs(from []byte, to [][]byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.ensureFrom(from)
	rm.ensureToAll(rm.GetRefsFrom(from))
	rm.ensureToAll(to)
	rm.RWGraph.SetRefs(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
// The remote state for all input keys will be read if not already read to ensure a consistent update.
func (rm *RemoteGraph) AddRefs(from []byte, to ...[]byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.ensureFrom(from)
	rm.ensureToAll(rm.GetRefsFrom(from))
	rm.ensureToAll(to)
	rm.RWGraph.AddRefs(from, to...)
}

// DeleteRefsFrom removes all references from the given input id.
func (rm *RemoteGraph) DeleteRefsFrom(from []byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.ensureFrom(from)
	rm.ensureToAll(rm.GetRefsFrom(from))
	rm.RWGraph.DeleteRefsFrom(from)
}

// DeleteRefsTo removes the input id from all ids that reference it.
func (rm *RemoteGraph) DeleteRefsTo(to []byte) {
	// Copy in the state needed to calculate the necessary updates, and apply the updates.
	rm.ensureTo(to)
	rm.ensureFromAll(rm.GetRefsTo(to))
	rm.RWGraph.DeleteRefsTo(to)
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

func (rm *RemoteGraph) ensureFromAll(froms [][]byte) {
	if len(froms) == 0 {
		return
	}

	unfetched := make([][]byte, 0, len(froms))
	for _, from := range froms {
		strFrom := string(from)
		if rm.readForward.Add(strFrom) {
			unfetched = append(unfetched, from)
		}
	}

	if len(unfetched) == 0 {
		return
	}

	// Read them all at once.
	rm.remoteToRead(func(g RGraph) {
		for _, from := range unfetched {
			if g.HasRefsFrom(from) {
				rm.RWGraph.setFrom(from, g.GetRefsFrom(from))
			}
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

	if len(unfetched) == 0 {
		return
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
