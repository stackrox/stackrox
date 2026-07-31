package networkbaseline

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stretchr/testify/assert"
)

func testPeer(id string, port uint32) Peer {
	return Peer{
		Entity: networkgraph.Entity{
			ID:   id,
			Type: storage.NetworkEntityInfo_DEPLOYMENT,
		},
		DstPort:  port,
		Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
	}
}

func TestPeerSet_AddAndHas(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("a", 443)
	p3 := testPeer("b", 80)

	s.Add(p1)
	s.Add(p2)
	s.Add(p3)

	assert.True(t, s.Has(p1))
	assert.True(t, s.Has(p2))
	assert.True(t, s.Has(p3))
	assert.Equal(t, 3, s.Len())

	// Duplicate add is a no-op.
	s.Add(p1)
	assert.Equal(t, 3, s.Len())
}

func TestPeerSet_Delete(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("a", 443)
	s.Add(p1)
	s.Add(p2)

	s.Delete(p1)
	assert.False(t, s.Has(p1))
	assert.True(t, s.Has(p2))
	assert.Equal(t, 1, s.Len())

	s.Delete(p2)
	assert.Equal(t, 0, s.Len())
	assert.True(t, s.IsEmpty())

	// Delete of absent peer is a no-op.
	s.Delete(p1)
	assert.Equal(t, 0, s.Len())
}

func TestPeerSet_GetByEntityID(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("a", 443)
	p3 := testPeer("b", 80)
	s.Add(p1)
	s.Add(p2)
	s.Add(p3)

	got := s.GetByEntityID("a")
	assert.ElementsMatch(t, []Peer{p1, p2}, got)

	got = s.GetByEntityID("b")
	assert.ElementsMatch(t, []Peer{p3}, got)

	got = s.GetByEntityID("nonexistent")
	assert.Empty(t, got)
}

func TestPeerSet_DeleteByEntityID(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("a", 443)
	p3 := testPeer("b", 80)
	s.Add(p1)
	s.Add(p2)
	s.Add(p3)

	s.DeleteByEntityID("a")
	assert.False(t, s.Has(p1))
	assert.False(t, s.Has(p2))
	assert.True(t, s.Has(p3))
	assert.Equal(t, 1, s.Len())
	assert.Empty(t, s.GetByEntityID("a"))
}

func TestPeerSet_ForEach(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("b", 443)
	s.Add(p1)
	s.Add(p2)

	var collected []Peer
	s.ForEach(func(p Peer) {
		collected = append(collected, p)
	})
	assert.ElementsMatch(t, []Peer{p1, p2}, collected)
}

func TestPeerSet_AsSlice(t *testing.T) {
	s := NewPeerSet()
	p1 := testPeer("a", 80)
	p2 := testPeer("a", 443)
	p3 := testPeer("b", 80)
	s.Add(p1)
	s.Add(p2)
	s.Add(p3)

	assert.ElementsMatch(t, []Peer{p1, p2, p3}, s.AsSlice())
}

func TestPeerSet_Empty(t *testing.T) {
	s := NewPeerSet()
	assert.True(t, s.IsEmpty())
	assert.Equal(t, 0, s.Len())
	assert.Empty(t, s.AsSlice())
	assert.Empty(t, s.GetByEntityID("any"))
}
