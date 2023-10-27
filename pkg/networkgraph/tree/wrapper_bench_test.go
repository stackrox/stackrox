package tree

import (
	"testing"

	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/require"
)

func BenchmarkNetworkTreeWrapper(b *testing.B) {
	b.Skip("ROX-20480: This test is failing. Skipping!")
	entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(32, 15000)
	require.NoError(b, err)
	ipv6Entities, err := testutils.GenRandomExtSrcNetworkEntityInfo(128, 5000)
	require.NoError(b, err)
	entities = append(entities, ipv6Entities...)

	var tree NetworkTree

	b.Run("NetworkTreeWrapper:Create", func(b *testing.B) {
		tree, err = NewNetworkTreeWrapper(entities)
		require.NoError(b, err)
	})

	b.Run("NetworkTreeWrapper:Get", func(b *testing.B) {
		for _, e := range entities {
			require.Equal(b, e, tree.Get(e.GetId()))
		}
	})

	b.Run("NetworkTreeWrapper:GetSupernet", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, tree.GetSupernet(e.GetId()))
		}
	})

	b.Run("NetworkTreeWrapper:GetSupernetForCIDR", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, tree.GetSupernetForCIDR(e.GetExternalSource().GetCidr()))
		}
	})

	b.Run("NetworkTreeWrapper:GetSubnets", func(b *testing.B) {
		for _, e := range entities {
			require.NotNil(b, tree.GetSubnets(e.GetId()))
		}
	})

	b.Run("NetworkTreeWrapper:GetSubnetsForCIDR", func(b *testing.B) {
		for _, e := range entities {
			tree.GetSubnetsForCIDR(e.GetExternalSource().GetCidr())
		}
	})

	b.Run("NetworkTreeWrapper:Insert", func(b *testing.B) {
		tree = NewDefaultNetworkTreeWrapper()
		for _, e := range entities {
			require.NoError(b, tree.Insert(e))
		}
	})
}
