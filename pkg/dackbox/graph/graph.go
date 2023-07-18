package graph

import (
	"bytes"

	"github.com/stackrox/rox/pkg/concurrency/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/utils"
	"github.com/stackrox/rox/pkg/sliceutils"
)

// RGraph is a read-only view of a Graph.
//
//go:generate mockgen-wrapper
type RGraph interface {
	HasRefsFrom(from []byte) bool
	HasRefsTo(to []byte) bool

	CountRefsFrom(from []byte) int
	CountRefsTo(to []byte) int

	GetRefsFrom(from []byte) [][]byte
	GetRefsTo(to []byte) [][]byte

	GetRefsFromPrefix(from, prefix []byte) [][]byte
	GetRefsToPrefix(to, prefix []byte) [][]byte

	CountRefsFromPrefix(from, prefix []byte) int
	CountRefsToPrefix(to, prefix []byte) int

	ReferencedFromPrefix(to, prefix []byte) bool
	ReferencesPrefix(from, prefix []byte) bool
}

// RWGraph is a read-write view of a Graph.
//
//go:generate mockgen-wrapper
type RWGraph interface {
	RGraph
	applyableGraph

	SetRefs(from []byte, to [][]byte)
	AddRefs(from []byte, to ...[]byte)

	DeleteRefsFrom(from []byte)
	DeleteRefsTo(from []byte)
}

type applyableGraph interface {
	setFrom(from []byte, to [][]byte)
	deleteFrom(from []byte)
	setTo(to []byte, from [][]byte)
	deleteTo(to []byte)
}

// DiscardableRGraph is an RGraph (read only view of the ID->[]ID map layer) that needs to be discarded when finished.
// NOTE: THIS HAS TO BE HERE FOR MOCK GENERATION TO WORK. IF YOU PUT IT IN A DIFFERENT FILE, 'go generate' WILL FAIL.
//
//go:generate mockgen-wrapper
type DiscardableRGraph interface {
	RGraph

	Discard()
}

// NewGraph is the basic type holding forward and backward ID relationships.
func NewGraph() *Graph {
	return &Graph{
		forward:  make(map[string][][]byte),
		backward: make(map[string][][]byte),
	}
}

// Graph holds forward and backward edge lists which can be modified by Add, Set, and Delete calls.
type Graph struct {
	forward  map[string][][]byte
	backward map[string][][]byte
}

// HasRefsFrom returns if there is an entry with 0 or more child keys in the graph.
func (s *Graph) HasRefsFrom(from []byte) bool {
	_, exists := s.forward[string(from)]
	return exists
}

// HasRefsTo returns if there is an entry with 0 or more parent keys in the graph.
func (s *Graph) HasRefsTo(to []byte) bool {
	_, exists := s.backward[string(to)]
	return exists
}

// CountRefsFrom returns the number of children reference from the input parent key.
func (s *Graph) CountRefsFrom(from []byte) int {
	return len(s.forward[string(from)])
}

// CountRefsTo returns the number of parents that reference the input child key.
func (s *Graph) CountRefsTo(to []byte) int {
	return len(s.backward[string(to)])
}

// GetRefsFrom returns the children referenced by the input parent key.
func (s *Graph) GetRefsFrom(from []byte) [][]byte {
	if keys, exist := s.forward[string(from)]; exist {
		return sliceutils.ShallowClone(keys)
	}
	return nil
}

// GetRefsTo returns the parents that reference the input child key.
func (s *Graph) GetRefsTo(to []byte) [][]byte {
	if keys, exist := s.backward[string(to)]; exist {
		return sliceutils.ShallowClone(keys)
	}
	return nil
}

// GetRefsFromPrefix returns the children referenced by the input parent key that have the passed prefix.
func (s *Graph) GetRefsFromPrefix(to, prefix []byte) [][]byte {
	if keys, exist := s.forward[string(to)]; exist {
		return sliceutils.ShallowClone(filterByPrefix(prefix, keys))
	}
	return nil
}

// GetRefsToPrefix returns the keys that have the passed prefix that reference the passed key
func (s *Graph) GetRefsToPrefix(to, prefix []byte) [][]byte {
	if keys, exist := s.backward[string(to)]; exist {
		return sliceutils.ShallowClone(filterByPrefix(prefix, keys))
	}
	return nil
}

// CountRefsFromPrefix returns the number of children referenced by the input parent key that have the passed prefix.
func (s *Graph) CountRefsFromPrefix(to, prefix []byte) int {
	if keys, exist := s.forward[string(to)]; exist {
		return len(filterByPrefix(prefix, keys))
	}
	return 0
}

// CountRefsToPrefix returns the number of keys that have the passed prefix that reference the passed key
func (s *Graph) CountRefsToPrefix(to, prefix []byte) int {
	if keys, exist := s.backward[string(to)]; exist {
		return len(filterByPrefix(prefix, keys))
	}
	return 0
}

// ReferencedFromPrefix returns whether a reference to the current key
// with the specified prefix exists in the graph.
func (s *Graph) ReferencedFromPrefix(to []byte, prefix []byte) bool {
	if keys, exists := s.backward[string(to)]; exists {
		return findFirstWithPrefix(prefix, keys) != -1
	}
	return false
}

// ReferencesPrefix returns whether a reference from the current key
// to a key with the specified prefix exists in the graph.
func (s *Graph) ReferencesPrefix(from, prefix []byte) bool {
	if keys, exists := s.forward[string(from)]; exists {
		return findFirstWithPrefix(prefix, keys) != -1
	}
	return false
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
func (s *Graph) SetRefs(from []byte, to [][]byte) {
	s.removeMappingsFrom(from)
	s.setMappings(from, to)
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
func (s *Graph) AddRefs(from []byte, to ...[]byte) {
	s.addMappings(from, to)
}

// DeleteRefsFrom removes all references from the input key.
func (s *Graph) DeleteRefsFrom(from []byte) {
	s.removeMappingsFrom(from)
}

// DeleteRefsTo removes all references to the input key.
func (s *Graph) DeleteRefsTo(to []byte) {
	s.removeMappingsTo(to)
}

// Copy creates a copy of the Graph.
func (s *Graph) Copy() *Graph {
	ret := &Graph{
		forward:  make(map[string][][]byte, len(s.forward)),
		backward: make(map[string][][]byte, len(s.backward)),
	}
	for from, tos := range s.forward {
		ret.forward[from] = utils.CopyKeys(tos)
	}
	for to, froms := range s.backward {
		ret.backward[to] = utils.CopyKeys(froms)
	}
	return ret
}

// Clear resets the Graph to be empty.
func (s *Graph) Clear() {
	s.forward = make(map[string][][]byte)
	s.backward = make(map[string][][]byte)
}

// Used by ModifiedGraph to set values without causing dependent changes.
/////////////////////////////////////////////////////////////////////////

func (s *Graph) setFrom(from []byte, to [][]byte) {
	s.forward[string(from)] = to
}

func (s *Graph) deleteFrom(from []byte) {
	delete(s.forward, string(from))
}

func (s *Graph) setTo(to []byte, from [][]byte) {
	s.backward[string(to)] = from
}

func (s *Graph) deleteTo(to []byte) {
	delete(s.backward, string(to))
}

// Helper functions.
////////////////////

func (s *Graph) addMappings(from []byte, to [][]byte) {
	sFrom := string(from)
	s.forward[sFrom] = sortedkeys.SortedKeys(s.forward[sFrom]).Union(sortedkeys.Sort(to))
	for _, t := range to {
		sTo := string(t)
		s.backward[sTo], _ = sortedkeys.SortedKeys(s.backward[sTo]).Insert(from)
	}
}

func (s *Graph) setMappings(from []byte, to [][]byte) {
	sFrom := string(from)
	s.forward[sFrom] = sortedkeys.Sort(to)
	for _, t := range to {
		sTo := string(t)
		s.backward[sTo], _ = sortedkeys.SortedKeys(s.backward[sTo]).Insert(from)
	}
}

func (s *Graph) removeMappingsFrom(from []byte) {
	sFrom := string(from)
	tos, exists := s.forward[sFrom]
	if !exists {
		return
	}
	for _, to := range tos {
		sTo := string(to)
		s.backward[sTo], _ = sortedkeys.SortedKeys(s.backward[sTo]).Remove(from)
		if len(s.backward[sTo]) == 0 {
			delete(s.backward, sTo)
		}
	}
	delete(s.forward, sFrom)
}

func (s *Graph) removeMappingsTo(to []byte) {
	sTo := string(to)
	froms, exists := s.backward[sTo]
	if !exists {
		return
	}
	for _, from := range froms {
		sFrom := string(from)
		s.forward[sFrom], _ = sortedkeys.SortedKeys(s.forward[sFrom]).Remove(to)
		if len(s.forward[sFrom]) == 0 {
			delete(s.forward, sFrom)
		}
	}
	delete(s.backward, sTo)
}

// findFirstWithPrefix determines the first index of a key that has the desired prefix. If no key
// has the desired prefix, it returns -1.
func findFirstWithPrefix(prefix []byte, keys [][]byte) int {
	if len(keys) == 0 {
		return -1
	}

	// If the prefix is lexicographically larger than the largest element in the slice, we can't have a match
	if bytes.Compare(prefix, keys[len(keys)-1]) > 0 {
		return -1
	}

	prefixLen := len(prefix)
	for i, key := range keys {
		if len(key) > prefixLen {
			key = key[:prefixLen]
		}
		if cmp := bytes.Compare(prefix, key); cmp == 0 {
			// we have a match
			return i
		} else if cmp < 0 {
			// prefix is lexicographically smaller already, and keys only get bigger - we can no longer have a match
			return -1
		}
	}
	return -1
}

// filterByPrefix returns a subslice (no copy) of the range within keys of all elements with the desired prefix.
// If no element in keys has the desired prefix, nil is returned.
func filterByPrefix(prefix []byte, keys [][]byte) [][]byte {
	low := findFirstWithPrefix(prefix, keys)
	if low == -1 {
		return nil
	}

	for high := len(keys) - 1; high >= low; high-- {
		// We no longer need to do a comparison, because we already have established that the last element is >= prefix,
		// and there exists an element (keys[low]) that has the desired prefix.
		if bytes.HasPrefix(keys[high], prefix) {
			return keys[low : high+1]
		}
	}
	return nil
}
