package net

import (
	"fmt"

	"github.com/stackrox/rox/generated/api/v1"
)

// L4Proto represents the L4 protocol (TCP, UDP etc).
type L4Proto int

// L4Proto constant values.
const (
	TCP L4Proto = iota
	UDP
	ICMP
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

// L4ProtoFromProtobuf translate a protobuf `data.L4Protocol` enum to an L4Proto.
func L4ProtoFromProtobuf(l4proto v1.L4Protocol) L4Proto {
	switch l4proto {
	case v1.L4Protocol_L4_PROTOCOL_TCP:
		return TCP
	case v1.L4Protocol_L4_PROTOCOL_UDP:
		return UDP
	case v1.L4Protocol_L4_PROTOCOL_ICMP:
		return ICMP
	default:
		return L4Proto(-1)
	}
}

// NumericEndpoint is an ip:port pair along with an L4 protocol.
type NumericEndpoint struct {
	IPAndPort IPPortPair
	L4Proto   L4Proto
}

// MakeNumericEndpoint returns a numeric endpoint for the given ip, port, and L4 protocol.
func MakeNumericEndpoint(addr IPAddress, port uint16, proto L4Proto) NumericEndpoint {
	return NumericEndpoint{
		IPAndPort: IPPortPair{
			Address: addr,
			Port:    port,
		},
		L4Proto: proto,
	}
}

// IsValid checks if the given numeric endpoint is valid.
func (e NumericEndpoint) IsValid() bool {
	return e.IPAndPort.IsValid()
}

// String returns a string representation of this numeric endpoint.
func (e NumericEndpoint) String() string {
	return fmt.Sprintf("%s (%s)", e.IPAndPort, e.L4Proto)
}
