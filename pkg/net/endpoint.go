package net

import (
	"fmt"
	"hash"

	"github.com/stackrox/rox/generated/storage"
)

// L4Proto represents the L4 protocol (TCP, UDP etc).
type L4Proto int

// L4Proto constant values.
const (
	TCP L4Proto = iota
	UDP
	ICMP
)

var (
	// ExternalIPv4Addr is the "canonical" external address sent by collector when the precise IPv4 address is not needed.
	ExternalIPv4Addr = ParseIP("255.255.255.255")
	// ExternalIPv6Addr is the "canonical" external address sent by collector when the precise IPv6 address is not needed.
	ExternalIPv6Addr = ParseIP("ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff")
)

// String represents a string representation of this L4 protocol.
func (p L4Proto) String() string {
	switch p {
	case TCP:
		return "tcp"
	case UDP:
		return "udp"
	case ICMP:
		return "icmp"
	default:
		return "<invalid l4 protocol>"
	}
}

// ToProtobuf translates this L4Proto to a protobuf `storage.L4Protocol` enum.
func (p L4Proto) ToProtobuf() storage.L4Protocol {
	switch p {
	case TCP:
		return storage.L4Protocol_L4_PROTOCOL_TCP
	case UDP:
		return storage.L4Protocol_L4_PROTOCOL_UDP
	case ICMP:
		return storage.L4Protocol_L4_PROTOCOL_ICMP
	default:
		return storage.L4Protocol_L4_PROTOCOL_RAW
	}
}

// L4ProtoFromProtobuf translate a protobuf `storage.L4Protocol` enum to an L4Proto.
func L4ProtoFromProtobuf(l4proto storage.L4Protocol) L4Proto {
	switch l4proto {
	case storage.L4Protocol_L4_PROTOCOL_TCP:
		return TCP
	case storage.L4Protocol_L4_PROTOCOL_UDP:
		return UDP
	case storage.L4Protocol_L4_PROTOCOL_ICMP:
		return ICMP
	default:
		return L4Proto(-1)
	}
}

// NumericEndpoint is an ip:port pair along with an L4 protocol.
type NumericEndpoint struct {
	IPAndPort NetworkPeerID
	L4Proto   L4Proto
}

// MakeNumericEndpoint returns a numeric endpoint for the given ip, port, and L4 protocol.
func MakeNumericEndpoint(addr IPAddress, port uint16, proto L4Proto) NumericEndpoint {
	return NumericEndpoint{
		IPAndPort: NetworkPeerID{
			Address: addr,
			Port:    port,
		},
		L4Proto: proto,
	}
}

// IsValid checks if the given numeric endpoint is valid.
func (e NumericEndpoint) IsValid() bool {
	return e.IPAndPort.IsAddressValid()
}

// String returns a string representation of this numeric endpoint.
func (e NumericEndpoint) String() string {
	return fmt.Sprintf("%s (%s)", e.IPAndPort, e.L4Proto)
}

// IsConsideredExternal checks whether the given numeric endpoint is considered as external IP by collector.
func (e NumericEndpoint) IsConsideredExternal() bool {
	return e.IPAndPort.Address == ExternalIPv4Addr || e.IPAndPort.Address == ExternalIPv6Addr
}

// NumericEndpointCompare returns -1;0;1 for a<b; a==b; a>b comparison by IP, port, then protocol. Used for slices.SortFunc.
func NumericEndpointCompare(a, b NumericEndpoint) int {
	cmp := IPAddressCompare(a.IPAndPort.Address, b.IPAndPort.Address)
	if cmp != 0 {
		return cmp
	}
	portCompare := int(a.IPAndPort.Port) - int(b.IPAndPort.Port)
	if portCompare != 0 {
		return portCompare
	}
	return int(a.L4Proto) - int(b.L4Proto)
}

// BinaryHash is a uint64 hash for memory-efficient map storage.
// Using hash keys instead of full NumericEndpoint structs (56 bytes) reduces
// map storage overhead and enables faster lookups via runtime.mapassign_fast64.
type BinaryHash uint64

var hashDelimiter = []byte{0}

// BinaryKey produces a binary hash for this endpoint.
// Uses xxhash for fast, non-cryptographic hashing with low collision probability.
// The buf parameter must be at least [16]byte to avoid allocations for IPv6 addresses.
func (e NumericEndpoint) BinaryKey(h hash.Hash64, buf *[16]byte) BinaryHash {
	h.Reset()

	// Hash IP address bytes by copying to buffer to avoid allocation
	if e.IPAndPort.Address.IsValid() {
		switch data := e.IPAndPort.Address.data.(type) {
		case ipv4data:
			copy(buf[:4], data[:])
			_, _ = h.Write(buf[:4])
		case ipv6data:
			copy(buf[:16], data[:])
			_, _ = h.Write(buf[:16])
		}
	}
	_, _ = h.Write(hashDelimiter)

	// Hash port (big-endian)
	buf[0] = byte(e.IPAndPort.Port >> 8)
	buf[1] = byte(e.IPAndPort.Port)
	_, _ = h.Write(buf[:2])

	// Hash protocol (big-endian)
	buf[0] = byte(e.L4Proto >> 56)
	buf[1] = byte(e.L4Proto >> 48)
	buf[2] = byte(e.L4Proto >> 40)
	buf[3] = byte(e.L4Proto >> 32)
	buf[4] = byte(e.L4Proto >> 24)
	buf[5] = byte(e.L4Proto >> 16)
	buf[6] = byte(e.L4Proto >> 8)
	buf[7] = byte(e.L4Proto)
	_, _ = h.Write(buf[:8])

	return BinaryHash(h.Sum64())
}
