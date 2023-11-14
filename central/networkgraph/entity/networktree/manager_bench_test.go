package networktree

import (
	"context"
	"net"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/networkgraph/externalsrcs"
	"github.com/stretchr/testify/require"
)

func BenchmarkCreateNetworkTree(b *testing.B) {
	ip := net.ParseIP("1.1.1.1")
	entities := make([]*storage.NetworkEntityInfo, 10000)
	const c1 = "c1"
	for i := range entities {
		cidr := ip.String() + "/32"
		id, _ := externalsrcs.NewClusterScopedID(c1, cidr)
		e := &storage.NetworkEntityInfo{
			Id:   id.String(),
			Type: storage.NetworkEntityInfo_EXTERNAL_SOURCE,
			Desc: &storage.NetworkEntityInfo_ExternalSource_{
				ExternalSource: &storage.NetworkEntityInfo_ExternalSource{
					Name: cidr,
					Source: &storage.NetworkEntityInfo_ExternalSource_Cidr{
						Cidr: cidr,
					},
				},
			},
		}
		entities[i] = e
		ip = nextIP(ip)
	}
	entitiesByCluster := map[string][]*storage.NetworkEntityInfo{c1: entities}

	// Above data will create one of the worst possible tree since each CIDR is disjoint aka all nodes are leaves,
	// hence resulting in comparison with every node for each new entry.

	b.Run("initialize", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			mgr := initialize(b, entitiesByCluster)
			require.NotNil(b, mgr)
		}
	})

	b.Run("insertIntoNetworkTree", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			mgr := initialize(b, entitiesByCluster)
			t := mgr.CreateNetworkTree(context.Background(), "c2")
			b.StartTimer()

			for _, entity := range entities {
				require.NoError(b, t.Insert(entity))
			}
		}
	})
}

func initialize(b *testing.B, entitiesByCluster map[string][]*storage.NetworkEntityInfo) Manager {
	mgr := newManager()
	err := mgr.Initialize(entitiesByCluster)
	require.NoError(b, err)
	return mgr
}

func nextIP(ip net.IP) net.IP {
	i := ip.To4()
	v := uint(i[0])<<24 + uint(i[1])<<16 + uint(i[2])<<8 + uint(i[3])
	v++
	v3 := byte(v & 0xFF)
	v2 := byte((v >> 8) & 0xFF)
	v1 := byte((v >> 16) & 0xFF)
	v0 := byte((v >> 24) & 0xFF)
	return net.IPv4(v0, v1, v2, v3)
}
