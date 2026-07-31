package networkbaseline

import "slices"

// PeerSet is a set of Peer values indexed by entity ID for efficient lookup.
type PeerSet struct {
	byEntityID map[string][]Peer
	size       int
}

// NewPeerSet creates an empty PeerSet.
func NewPeerSet() *PeerSet {
	return &PeerSet{byEntityID: make(map[string][]Peer)}
}

// Add inserts a peer into the set. Duplicate peers are ignored.
func (s *PeerSet) Add(p Peer) {
	peers := s.byEntityID[p.Entity.ID]
	for _, existing := range peers {
		if existing == p {
			return
		}
	}
	s.byEntityID[p.Entity.ID] = append(peers, p)
	s.size++
}

// Delete removes a peer from the set. No-op if not present.
func (s *PeerSet) Delete(p Peer) {
	peers := s.byEntityID[p.Entity.ID]
	for i, existing := range peers {
		if existing == p {
			s.byEntityID[p.Entity.ID] = slices.Delete(peers, i, i+1)
			s.size--
			if len(s.byEntityID[p.Entity.ID]) == 0 {
				delete(s.byEntityID, p.Entity.ID)
			}
			return
		}
	}
}

// Has reports whether the set contains the given peer.
func (s *PeerSet) Has(p Peer) bool {
	for _, existing := range s.byEntityID[p.Entity.ID] {
		if existing == p {
			return true
		}
	}
	return false
}

// GetByEntityID returns all peers with the given entity ID.
func (s *PeerSet) GetByEntityID(id string) []Peer {
	return s.byEntityID[id]
}

// DeleteByEntityID removes all peers with the given entity ID.
func (s *PeerSet) DeleteByEntityID(id string) {
	s.size -= len(s.byEntityID[id])
	delete(s.byEntityID, id)
}

// ForEach calls fn for each peer in the set.
func (s *PeerSet) ForEach(fn func(Peer)) {
	for _, peers := range s.byEntityID {
		for _, p := range peers {
			fn(p)
		}
	}
}

// Len returns the number of peers in the set.
func (s *PeerSet) Len() int {
	return s.size
}

// IsEmpty reports whether the set contains no peers.
func (s *PeerSet) IsEmpty() bool {
	return s.size == 0
}

// AsSlice returns all peers as a flat slice.
func (s *PeerSet) AsSlice() []Peer {
	out := make([]Peer, 0, s.size)
	for _, peers := range s.byEntityID {
		out = append(out, peers...)
	}
	return out
}
