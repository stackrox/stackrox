package sortutils

import (
	"bytes"

	pkgNet "github.com/stackrox/rox/pkg/net"
)

// SortableIPv4NetworkSlice allows us to sort the IPv4 networks in ascending lexical byte order. Since, the host
// identifier bits (bits not in network prefix) are all set to 0, this gives us largest to smallest network ordering.
// e.g. 10.0.0.0/8, 10.0.0.0/24, 10.10.0.0/24, 127.0.0.0/8, ...
type SortableIPv4NetworkSlice []pkgNet.IPNetwork

func (s SortableIPv4NetworkSlice) Len() int {
	return len(s)
}

func (s SortableIPv4NetworkSlice) Less(i, j int) bool {
	if s[i].PrefixLen() != s[j].PrefixLen() {
		return s[i].PrefixLen() < s[j].PrefixLen()
	}
	if !s[i].IP().AsNetIP().Equal(s[j].IP().AsNetIP()) {
		return bytes.Compare(s[i].IP().AsNetIP(), s[j].IP().AsNetIP()) > 0
	}
	return false
}

func (s SortableIPv4NetworkSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// SortableIPv6NetworkSlice allows us to sort the IPv6 networks in ascending lexical byte order. Since, the host
// identifier bits (bits not in network prefix) are all set to 0, this gives us largest to smallest network ordering.
// e.g. ipv6(10.0.0.0/8), ipv6(10.0.0.0/24), ipv6(10.10.0.0/24), ipv6(127.0.0.0/8), ...
type SortableIPv6NetworkSlice []pkgNet.IPNetwork

func (s SortableIPv6NetworkSlice) Len() int {
	return len(s)
}

func (s SortableIPv6NetworkSlice) Less(i, j int) bool {
	if s[i].PrefixLen() != s[j].PrefixLen() {
		return s[i].PrefixLen() < s[j].PrefixLen()
	}
	if !s[i].IP().AsNetIP().Equal(s[j].IP().AsNetIP()) {
		return bytes.Compare(s[i].IP().AsNetIP(), s[j].IP().AsNetIP()) > 0
	}
	return false
}

func (s SortableIPv6NetworkSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
