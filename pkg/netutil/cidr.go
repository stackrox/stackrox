package netutil

import (
	"net"

	"github.com/stackrox/rox/pkg/utils"
)

// MustParseCIDR parses the given CIDR string and returns the corresponding IPNet. If the string is invalid, this
// function panics.
func MustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	utils.Must(err)
	return ipNet
}

// IsIPNetSubset checks if maybeSubset is fully contained within ipNet.
func IsIPNetSubset(ipNet *net.IPNet, maybeSubset *net.IPNet) bool {
	if !ipNet.Contains(maybeSubset.IP) {
		return false
	}
	if len(ipNet.Mask) != len(maybeSubset.Mask) {
		return false
	}

	for i, byte := range ipNet.Mask {
		if byte&maybeSubset.Mask[i] != byte {
			return false
		}
	}
	return true
}
