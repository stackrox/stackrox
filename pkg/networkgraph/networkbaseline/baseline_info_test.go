package networkbaseline

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

func makePeerSet(n int) *PeerSet {
	s := NewPeerSet()
	for i := range n {
		s.Add(Peer{
			Entity: networkgraph.Entity{
				ID:   fmt.Sprintf("deploy-%d", i),
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
			},
			DstPort:  uint32(8080 + i%10),
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		})
	}
	return s
}

func BenchmarkGetPeer(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("peers=%d", size), func(b *testing.B) {
			ps := makePeerSet(size)
			b.ResetTimer()
			for range b.N {
				ps.GetByEntityID("nonexistent")
			}
		})
	}
}
