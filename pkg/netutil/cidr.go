package netutil

import (
	"net"

	"github.com/stackrox/rox/pkg/utils"
)

var (
	privateSubnets = []*net.IPNet{
		MustParseCIDR("127.0.0.0/8"),    // IPv4 localhost
		MustParseCIDR("10.0.0.0/8"),     // class A
		MustParseCIDR("172.16.0.0/12"),  // class B
		MustParseCIDR("192.168.0.0/16"), // class C
		MustParseCIDR("::1/128"),        // IPv6 localhost
		MustParseCIDR("fd00::/8"),       // IPv6 ULA
	}
)

// IsCIDRBlockInPrivateSubnet parses cidrStr and checks if it falls under the RFC 1819 private IP range
func IsCIDRBlockInPrivateSubnet(cidrStr string) bool {
	_, cidr, err := net.ParseCIDR(cidrStr)
	utils.CrashOnError(err)
	for _, subnet := range privateSubnets {
		if IsIPNetSubset(subnet, cidr) {
			return true
		}
	}
	return false
}


// MustParseCIDR parses the given CIDR string and returns the corresponding IPNet. If the string is invalid, this
// function panics.
func MustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	utils.CrashOnError(err)
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
