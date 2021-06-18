package tree

import (
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph/testutils"
	"github.com/stretchr/testify/require"
)

func BenchmarkNRadixTreeForIPv4(b *testing.B) {
	entities, err := getNetworkEntities(32, 20000)
	require.NoError(b, err)

	runBenchMarkOnOps(b, pkgNet.IPv4, entities, "IPv4NRadixTree")
}

func BenchmarkNRadixTreeForIPv6(b *testing.B) {
	entities, err := getNetworkEntities(128, 20000)
	require.NoError(b, err)

	runBenchMarkOnOps(b, pkgNet.IPv6, entities, "IPv6NRadixTree")
}

func getNetworkEntities(bits, numNetworks int) ([]*storage.NetworkEntityInfo, error) {
	nets := make(map[string]struct{})
	ipRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	maskRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	ip := make([]byte, bits/8)
	for len(nets) < numNetworks {
		if _, err := ipRand.Read(ip); err != nil {
			return nil, err
		}
		ipAddr := net.IP(ip)

		_, n, err := net.ParseCIDR(ipAddr.String() + "/" + strconv.Itoa(int(1+maskRand.Int31n(int32(bits)))))
		if err != nil {
			return nil, err
		}
		nets[n.String()] = struct{}{}
	}

	entities := make([]*storage.NetworkEntityInfo, 0, len(nets))
	for k := range nets {
		entities = append(entities, testutils.GetExtSrcNetworkEntityInfo(k, k, k, false))
	}

	return entities, nil
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
