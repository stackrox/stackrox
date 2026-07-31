package networkbaseline

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph"
)

func makePeers(n int) map[Peer]struct{} {
	peers := make(map[Peer]struct{}, n)
	for i := range n {
		peers[Peer{
			Entity: networkgraph.Entity{
				ID:   fmt.Sprintf("deploy-%d", i),
				Type: storage.NetworkEntityInfo_DEPLOYMENT,
			},
			DstPort:  uint32(8080 + i%10),
			Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
		}] = struct{}{}
	}
	return peers
}

func BenchmarkGetPeer(b *testing.B) {
	for _, size := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("peers=%d", size), func(b *testing.B) {
			info := &BaselineInfo{
				BaselinePeers: makePeers(size),
			}
			// Look up an ID that doesn't exist to measure worst-case (full scan).
			b.ResetTimer()
			for range b.N {
				info.GetPeer("nonexistent")
			}
		})
	}
}
