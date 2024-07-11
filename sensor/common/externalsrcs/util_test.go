package externalsrcs

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPNetworkSort(t *testing.T) {
	ipv4Slice := []byte{
		120, 0, 0, 0, 8,
		192, 16, 0, 0, 16,
		192, 0, 0, 0, 8,
		192, 0, 0, 0, 11,
	}

	sort.Sort(sortableIPv4NetworkSlice(ipv4Slice))

	expectedSortedSlice := []byte{
		192, 16, 0, 0, 16,
		192, 0, 0, 0, 11,
		192, 0, 0, 0, 8,
		120, 0, 0, 0, 8,
	}

	assert.Equal(t, expectedSortedSlice, ipv4Slice)

	ipv6Slice := []byte{
		0, 0, 0, 0, 0, 255, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
		0, 0, 0, 0, 0, 254, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
		0, 0, 0, 0, 0, 255, 128, 0, 0, 1, 3, 2, 2, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
	}

	sort.Sort(sortableIPv6NetworkSlice(ipv6Slice))

	expectedSortedSlice = []byte{
		0, 0, 0, 0, 0, 255, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
		0, 0, 0, 0, 0, 255, 128, 0, 0, 1, 3, 2, 2, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 254, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
		0, 0, 0, 0, 0, 0, 128, 0, 0, 1, 3, 2, 2, 5, 6, 6, 1,
	}

	assert.Equal(t, expectedSortedSlice, ipv6Slice)
}
