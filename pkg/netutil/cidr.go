package netutil

import (
	"net"

	"github.com/stackrox/rox/pkg/utils"
)

// MustParseCIDR parses the given CIDR string and returns the corresponding IPNet. If the string is invalid, this
// function panics.
func MustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	utils.CrashOnError(err)
	return ipNet
}

// IsIPNetOverlapingPrivateRange checks if network overlaps with private subnets
func IsIPNetOverlapingPrivateRange(ipNet *net.IPNet) bool {
	var privateSubnets []*net.IPNet
	privateSubnets = append(privateSubnets, IPv4PrivateNetworks...)
	privateSubnets = append(privateSubnets, IPv6PrivateNetworks...)
	return AnyOverlap(ipNet, privateSubnets)
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

// Overlap checks if two networks overlap.
func Overlap(n1, n2 *net.IPNet) bool {
	if len(n1.Mask) != len(n2.Mask) {
		return false
	}
	return n1.Contains(n2.IP) || n2.Contains(n1.IP)
}

// AnyOverlap checks if any network in ns overlaps with n1.
func AnyOverlap(n1 *net.IPNet, ns []*net.IPNet) bool {
	for _, n := range ns {
		if Overlap(n1, n) {
			return true
		}
	}
	return false
}
