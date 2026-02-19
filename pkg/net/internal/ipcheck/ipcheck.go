package ipcheck

import (
	"encoding/binary"

	"github.com/stackrox/rox/pkg/netutil"
)

// IPv4 private network masks and prefixes (generated from netutil.IPv4PrivateNetworks)
// This maintains a single source of truth while providing optimized constants
var (
	ipv4Masks    []uint32
	ipv4Prefixes []uint32
)

func init() {
	// Generate masks and prefixes from the canonical definitions in netutil
	ipv4Masks = make([]uint32, 0, len(netutil.IPv4PrivateNetworks))
	ipv4Prefixes = make([]uint32, 0, len(netutil.IPv4PrivateNetworks))
	for _, ipNet := range netutil.IPv4PrivateNetworks {
		// Extract mask as uint32
		maskBytes := ipNet.Mask
		if len(maskBytes) != 4 {
			panic("IPv4 network has invalid mask length")
		}
		ipv4Masks = append(ipv4Masks, binary.BigEndian.Uint32(maskBytes))

		// Extract network prefix as uint32
		ipBytes := ipNet.IP.To4()
		if ipBytes == nil {
			panic("IPv4 network has invalid IP")
		}
		ipv4Prefixes = append(ipv4Prefixes, binary.BigEndian.Uint32(ipBytes))
	}
}

// IsIPv4Public returns true if the IPv4 address is public (not in private ranges).
// Input is 4-byte array representing IPv4 address.
func IsIPv4Public(ip [4]byte) bool {
	// Convert to uint32 in network byte order (big-endian)
	ipInt := binary.BigEndian.Uint32(ip[:])

	// Check each private network range
	for i := range len(ipv4Masks) {
		if (ipInt & ipv4Masks[i]) == ipv4Prefixes[i] {
			return false // Is private
		}
	}

	return true // Is public
}

// IsIPv6Public returns true if the IPv6 address is public (not in private ranges).
// Input is 16-byte array representing IPv6 address.
func IsIPv6Public(ip [16]byte) bool {
	// Check fd00::/8 (Unique Local Address)
	if ip[0] == 0xfd {
		return false
	}

	// Check fe80::/10 (Link-Local)
	if ip[0] == 0xfe && (ip[1]&0xc0) == 0x80 {
		return false
	}

	// Check IPv4-mapped IPv6 (::ffff:0:0/96)
	if ip[0] == 0 && ip[1] == 0 && ip[2] == 0 && ip[3] == 0 &&
		ip[4] == 0 && ip[5] == 0 && ip[6] == 0 && ip[7] == 0 &&
		ip[8] == 0 && ip[9] == 0 && ip[10] == 0xff && ip[11] == 0xff {
		// Extract IPv4 part and check if private
		var ipv4 [4]byte
		copy(ipv4[:], ip[12:16])
		return IsIPv4Public(ipv4)
	}

	return true // Is public
}
