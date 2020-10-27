package tree

import (
	"sort"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stackrox/rox/pkg/networkgraph"
	"github.com/stackrox/rox/pkg/networkgraph/test"
	"github.com/stretchr/testify/assert"
)

func TestNetworkTree(t *testing.T) {
	/*
		Expected Tree:

		INTERNET
		 	|______ 3
			|		|__ 2
			|			|__ (1)
			|				 |__ 4
			|______ (6)
					|__ 5

	*/

	e1 := test.GetExtSrcNetworkEntity("1", "1", "35.187.144.0/20", true)
	e2 := test.GetExtSrcNetworkEntity("2", "2", "35.187.144.0/16", false)
	e3 := test.GetExtSrcNetworkEntity("3", "3", "35.187.144.0/8", false)
	e4 := test.GetExtSrcNetworkEntity("4", "4", "35.187.144.0/23", false)
	e5 := test.GetExtSrcNetworkEntity("5", "5", "36.188.144.0/30", false)
	e6 := test.GetExtSrcNetworkEntity("6", "6", "36.188.144.0/16", true)

	networkTree, err := NewNetworkTree([]*storage.NetworkEntityInfo{e1, e2, e3, e4, e5, e6}, pkgNet.IPv4)
	assert.NoError(t, err)

	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e4}, networkTree.GetSubnets("1"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e2}, networkTree.GetSubnets("3"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{}, networkTree.GetSubnets("4"))
	assert.Equal(t, e2, networkTree.GetSupernet("1"))
	assert.Equal(t, e6, networkTree.GetSupernet("5"))
	assert.Equal(t, networkgraph.InternetEntity().ToProto(), networkTree.GetSupernet(networkgraph.InternetExternalSourceID))

	e7 := test.GetExtSrcNetworkEntity("7", "7", "36.188.144.0/31", false)
	e8 := test.GetExtSrcNetworkEntity("8", "8", "35.188.144.0/5", false)

	assert.NoError(t, networkTree.Insert(e7))
	assert.NoError(t, networkTree.Insert(e8))

	/*
		Expected tree after inserts:

		INTERNET
			|_______ 8*
					 |
					 |______ 3
					 |		|__ 2
					 |			|__ (1)
					 |				 |__ 4
					 |______ (6)
							 |__ 5
							 	 |__7*

	*/
	assert.Equal(t, e5, networkTree.GetSupernet("7"))
	assert.Equal(t, e8, networkTree.GetSupernet("3"))
	assert.Equal(t, e8, networkTree.GetSupernet("6"))
	assert.ElementsMatch(t, []*storage.NetworkEntityInfo{e8}, networkTree.GetSubnets(networkgraph.InternetExternalSourceID))

	// Invalid CIDR
	assert.Error(t, networkTree.Insert(test.GetExtSrcNetworkEntity("9", "9", "0:0:0:0:0:ffff:0:0/0", false)))

	// Update CIDR
	e4 = test.GetExtSrcNetworkEntity("4", "4", "35.187.144.0/26", false)
	assert.NoError(t, networkTree.Insert(e4))
	assert.Equal(t, e4, networkTree.Get("4"))

	// Existing CIDR
	assert.Error(t, networkTree.Insert(test.GetExtSrcNetworkEntity("88", "88", "35.188.144.0/5", false)))

	// Skip default networks
	assert.Equal(t, e2, networkTree.GetMatchingSupernet("4", func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() }))

	// Only default networks
	assert.Equal(t, e6, networkTree.GetMatchingSupernet("7", func(e *storage.NetworkEntityInfo) bool { return e.GetExternalSource().GetDefault() }))

	networkTree.Remove("1")
	assert.Nil(t, networkTree.Get("1"))

	/*
		Expected tree after remove:

		INTERNET
			|_______ 8*
					 |
					 |______ 3
					 |		|__ 2
					 |			|__ 4
					 |
					 |______ (6)
							 |__ 5
							 	 |__7*

	*/
	assert.Equal(t, e2, networkTree.GetMatchingSupernet("4", func(e *storage.NetworkEntityInfo) bool { return !e.GetExternalSource().GetDefault() }))
}

func TestIPNetworkSort(t *testing.T) {
	ipv4Slice := []pkgNet.IPNetwork{
		pkgNet.IPNetworkFromCIDR("120.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.16.0.0/16"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/11"),
	}

	sort.Sort(sortableIPv4NetworkSlice(ipv4Slice))

	expectedSortedSlice := []pkgNet.IPNetwork{
		pkgNet.IPNetworkFromCIDR("192.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("120.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/11"),
		pkgNet.IPNetworkFromCIDR("192.16.0.0/16"),
	}

	assert.Equal(t, expectedSortedSlice, ipv4Slice)
}
