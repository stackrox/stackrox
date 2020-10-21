package networkgraph

import (
	"sort"
	"testing"

	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/assert"
)

func TestNetworkTree(t *testing.T) {

	/*
		INTERNET
		 	|______ 3
			|		|__ 2
			|			 |__ 1
			|				 |__ 4
			|______ 6
					|__ 5

	*/
	nets := map[string]string{
		"1": "35.187.144.0/20",
		"2": "35.187.144.0/16",
		"3": "35.187.144.0/8",
		"4": "35.187.144.0/23",
		"5": "36.188.144.0/30",
		"6": "36.188.144.0/16",
	}

	tree, err := NewNetworkTree(nets, pkgNet.IPv4)
	assert.NoError(t, err)

	keys := tree.GetSubnets("3")
	assert.Equal(t, []string{"2"}, keys)

	keys = tree.GetSubnets("1")
	assert.Equal(t, []string{"4"}, keys)

	keys = tree.GetSubnets("4")
	assert.Equal(t, []string{}, keys)

	key := tree.GetSupernet("5")
	assert.Equal(t, "6", key)

	key = tree.GetSupernet(InternetExternalSourceID)
	assert.Equal(t, InternetExternalSourceID, key)

	key = tree.GetSupernet("1")
	assert.Equal(t, "2", key)

	/*
		Tree rearrangement as a result of new inserts.

		INTERNET
			|_______ 8*
					 |
					 |______ 3
					 |		|__ 2
					 |			 |__ 1
					 |				 |__ 4
					 |______ 6
							 |__ 5
							 	 |__7*

	*/

	err = tree.Insert("7", "36.188.144.0/31")
	assert.NoError(t, err)

	err = tree.Insert("8", "35.188.144.0/5")
	assert.NoError(t, err)

	key = tree.GetSupernet("7")
	assert.Equal(t, "5", key)

	key = tree.GetSupernet("3")
	assert.Equal(t, "8", key)

	key = tree.GetSupernet("6")
	assert.Equal(t, "8", key)

	keys = tree.GetSubnets(InternetExternalSourceID)
	assert.Equal(t, []string{"8"}, keys)

	// Invalid CIDR
	err = tree.Insert("9", "0:0:0:0:0:ffff:0:0/0")
	assert.Error(t, err)

	// Update CIDR
	err = tree.Insert("4", "35.187.144.0/26")
	assert.NoError(t, err)
	cidr := tree.GetCIDR("4")
	assert.Equal(t, "35.187.144.0/26", cidr)

	// Existing CIDR
	err = tree.Insert("88", "35.188.144.0/5")
	assert.Error(t, err)
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
