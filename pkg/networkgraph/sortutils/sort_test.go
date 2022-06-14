package sortutils

import (
	"sort"
	"testing"

	pkgNet "github.com/stackrox/rox/pkg/net"
	"github.com/stretchr/testify/assert"
)

func TestIPNetworkSort(t *testing.T) {
	ipv4Slice := []pkgNet.IPNetwork{
		pkgNet.IPNetworkFromCIDR("120.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.16.0.0/16"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/11"),
	}

	sort.Sort(SortableIPv4NetworkSlice(ipv4Slice))

	expectedSortedSlice := []pkgNet.IPNetwork{
		pkgNet.IPNetworkFromCIDR("192.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("120.0.0.0/8"),
		pkgNet.IPNetworkFromCIDR("192.0.0.0/11"),
		pkgNet.IPNetworkFromCIDR("192.16.0.0/16"),
	}

	assert.Equal(t, expectedSortedSlice, ipv4Slice)
}
