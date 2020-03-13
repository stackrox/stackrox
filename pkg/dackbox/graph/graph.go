package graph

import (
	"github.com/stackrox/rox/pkg/dackbox/sortedkeys"
	"github.com/stackrox/rox/pkg/dackbox/utils"
)

// RGraph is a read-only view of a Graph.
//go:generate mockgen-wrapper
type RGraph interface {
	HasRefsFrom(from []byte) bool
	HasRefsTo(to []byte) bool

	CountRefsFrom(from []byte) int
	CountRefsTo(to []byte) int

	GetRefsFrom(from []byte) [][]byte
	GetRefsTo(to []byte) [][]byte
}

// RWGraph is a read-write view of a Graph.
//go:generate mockgen-wrapper
type RWGraph interface {
	RGraph
	applyableGraph

	SetRefs(from []byte, to [][]byte) error
	AddRefs(from []byte, to ...[]byte) error

	DeleteRefsFrom(from []byte) error
	DeleteRefsTo(from []byte) error
}

type applyableGraph interface {
	setFrom(from []byte, to [][]byte)
	deleteFrom(from []byte)
	setTo(to []byte, from [][]byte)
	deleteTo(to []byte)
}

// DiscardableRGraph is an RGraph (read only view of the ID->[]ID map layer) that needs to be discarded when finished.
// NOTE: THIS HAS TO BE HERE FOR MOCK GENERATION TO WORK. IF YOU PUT IT IN A DIFFERENT FILE, 'go generate' WILL FAIL.
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
		return append([][]byte{}, keys...)
	}
	return nil
}

// GetRefsTo returns the parents that reference the input child key.
func (s *Graph) GetRefsTo(to []byte) [][]byte {
	if keys, exist := s.backward[string(to)]; exist {
		return append([][]byte{}, keys...)
	}
	return nil
}

// SetRefs sets the children of 'from' to be the input list of keys 'to'.
func (s *Graph) SetRefs(from []byte, to [][]byte) error {
	s.removeMappingsFrom(from)
	s.setMappings(from, to)
	return nil
}

// AddRefs adds the set of keys 'to' to the list of children of 'from'.
func (s *Graph) AddRefs(from []byte, to ...[]byte) error {
	s.addMappings(from, to)
	return nil
}

// DeleteRefsFrom removes all references from the input key.
func (s *Graph) DeleteRefsFrom(from []byte) error {
	s.removeMappingsFrom(from)
	return nil
}

// DeleteRefsTo removes all references to the input key.
func (s *Graph) DeleteRefsTo(to []byte) error {
	s.removeMappingsTo(to)
	return nil
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
