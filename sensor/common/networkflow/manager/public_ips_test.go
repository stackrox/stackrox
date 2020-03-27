package manager

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPv6Sort(t *testing.T) {
	ipv6Slice := []uint64{
		14, 2,
		3, 100,
		100, 3,
		1, 1000,
		14, 3,
		14, 1,
	}

	sort.Sort(sortableIPv6Slice(ipv6Slice))

	expectedSortedSlice := []uint64{
		1, 1000,
		3, 100,
		14, 1,
		14, 2,
		14, 3,
		100, 3,
	}

	assert.Equal(t, expectedSortedSlice, ipv6Slice)
}
