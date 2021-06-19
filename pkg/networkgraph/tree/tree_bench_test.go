package tree

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/require"
)

func BenchmarkLegacyNetworkTreeForIPv4(b *testing.B) {
	entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(pkgNet.IPv4, 20000)
	require.NoError(b, err)

	runBenchmarkOnOpsOnLegacy(b, pkgNet.IPv4, entities, "IPv4LegacyNTree")
}

func BenchmarkLegacyNetworkTreeForIPv6(b *testing.B) {
	entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(pkgNet.IPv6, 20000)
	require.NoError(b, err)

	runBenchmarkOnOpsOnLegacy(b, pkgNet.IPv6, entities, "IPv6LegacyNTree")
}

func runBenchmarkOnOpsOnLegacy(b *testing.B, family pkgNet.Family, entities []*storage.NetworkEntityInfo, tcPrefix string) {
	var legacyT NetworkTree
	var err error

	b.Run(tcPrefix+":Create", func(b *testing.B) {
		legacyT, err = NewNetworkTree(family, entities)
		require.NoError(b, err)
	})

	b.Run(tcPrefix+":GetSupernet", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, legacyT.GetSupernet(e.GetId()))
		}
	})

	b.Run(tcPrefix+":GetSupernetForCIDR", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, legacyT.GetSupernetForCIDR(e.GetExternalSource().GetCidr()))
		}
	})

	b.Run(tcPrefix+":GetSubnets", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, legacyT.GetSubnets(e.GetId()))
		}
	})

	b.Run(tcPrefix+":GetSubnetsForCIDR", func(b *testing.B) {
		for _, e := range entities {
			legacyT.GetSubnetsForCIDR(e.GetExternalSource().GetCidr())
		}
	})

	b.Run(tcPrefix+":Insert", func(b *testing.B) {
		legacyT = NewDefaultNetworkTree(family)
		for _, e := range entities {
			require.NoError(b, legacyT.Insert(e))
		}
	})
}
