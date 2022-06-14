package tree

import (
	"testing"

	"github.com/stackrox/stackrox/generated/storage"
	pkgNet "github.com/stackrox/stackrox/pkg/net"
	"github.com/stackrox/stackrox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/require"
)

func BenchmarkNRadixTreeForIPv4(b *testing.B) {
	entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(pkgNet.IPv4, 20000)
	require.NoError(b, err)

	runBenchMarkOnOps(b, pkgNet.IPv4, entities, "IPv4NRadixTree")
}

func BenchmarkNRadixTreeForIPv6(b *testing.B) {
	entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(pkgNet.IPv6, 20000)
	require.NoError(b, err)

	runBenchMarkOnOps(b, pkgNet.IPv6, entities, "IPv6NRadixTree")
}

func runBenchMarkOnOps(b *testing.B, family pkgNet.Family, entities []*storage.NetworkEntityInfo, tcPrefix string) {
	var radixT NetworkTree
	var err error

	b.Run(tcPrefix+":Create", func(b *testing.B) {
		radixT, err = NewNRadixTree(family, entities)
		require.NoError(b, err)
	})

	b.Run(tcPrefix+":Get", func(b *testing.B) {
		for _, e := range entities {
			require.Equal(b, e, radixT.Get(e.GetId()))
		}
	})

	b.Run(tcPrefix+":GetSupernet", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, radixT.GetSupernet(e.GetId()))
		}
	})

	b.Run(tcPrefix+":GetSupernetForCIDR", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, radixT.GetSupernetForCIDR(e.GetExternalSource().GetCidr()))
		}
	})

	b.Run(tcPrefix+":GetSubnets", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, radixT.GetSubnets(e.GetId()))
		}
	})

	b.Run(tcPrefix+":GetSubnetsForCIDR", func(b *testing.B) {
		for _, e := range entities {
			radixT.GetSubnetsForCIDR(e.GetExternalSource().GetCidr())
		}
	})

	b.Run(tcPrefix+":Insert", func(b *testing.B) {
		radixT = NewDefaultNRadixTree(family)
		for _, e := range entities {
			require.NoError(b, radixT.Insert(e))
		}
	})
}
