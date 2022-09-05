package netutil

import "net"

var (
	// IPV4PrivateNetworks holds RFC1918 addresses plus other reserved IPv4 ranges.
	IPV4PrivateNetworks = []*net.IPNet{
		// private networks per RFC1918
		MustParseCIDR("10.0.0.0/8"),
		MustParseCIDR("172.16.0.0/12"),
		MustParseCIDR("192.168.0.0/16"),

		// Other reserved ranges
		MustParseCIDR("100.64.0.0/10"),
		MustParseCIDR("169.254.0.0/16"),
	}

	// IPV4LocalHost is the local host IP range 127.0.0.0/8.
	IPV4LocalHost = MustParseCIDR("127.0.0.0/8")

	// IPV6PrivateNetworks holds IPv6 private range and IPv4-mapped private addresses in IPv6 range.
	IPV6PrivateNetworks = []*net.IPNet{
		// Unique Local Addresses (ULA)
		MustParseCIDR("fd00::/8"),

		// IPv4-mapped IPv6 for private networks per RFC1918.
		MustParseCIDR("::ffff:10.0.0.0/104"),
		MustParseCIDR("::ffff:172.16.0.0/108"),
		MustParseCIDR("::ffff:192.168.0.0/112"),

		// Other reserved IPv4 ranges
		MustParseCIDR("::ffff:100.64.0.0/106"),
		MustParseCIDR("::ffff:169.254.0.0/112"),
	}

	// IPV6LocalHost is the local host IP range ::1/128.
	IPV6LocalHost = MustParseCIDR("::1/128")

	// IPV4MappedIPv6Loopback is the IPv4 loopback address mapped in IPv6 range.
	IPV4MappedIPv6Loopback = MustParseCIDR("::ffff:127.0.0.1/104")
)

// GetPrivateSubnets returns a slice of IPv4 and IPv6 addresses considered as private ranges including localhost addresses.
func GetPrivateSubnets() []*net.IPNet {
	var subnets []*net.IPNet
	subnets = append(subnets, IPV4PrivateNetworks...)
	subnets = append(subnets, IPV4LocalHost)
	subnets = append(subnets, IPV6PrivateNetworks...)
	subnets = append(subnets, IPV6LocalHost)
	return subnets
}
