package net

import (
	"bytes"
	"net"
)

// Family represents the address family of an IP address.
type Family int

const (
	// InvalidFamily represents the invalid IP address family value.
	InvalidFamily Family = iota
	// IPv4 represents the IPv4 address family.
	IPv4
	// IPv6 represents the IPv6 address family.
	IPv6
)

// String returns a string representation of the family
func (f Family) String() string {
	switch f {
	case IPv4:
		return "IPv4"
	case IPv6:
		return "IPv6"
	default:
		return "unknown"
	}
}

type ipAddrData interface {
	family() Family
	bytes() []byte
	isLoopback() bool
}

type ipv4data [4]byte

func (d ipv4data) family() Family {
	return IPv4
}
func (d ipv4data) bytes() []byte {
	return d[:]
}
func (d ipv4data) isLoopback() bool {
	// IPv4 loopback is 127.0.0.0/8
	return d[0] == 127
}

type ipv6data [16]byte

func (d ipv6data) family() Family {
	return IPv6
}
func (d ipv6data) bytes() []byte {
	return d[:]
}
func (d ipv6data) isLoopback() bool {
	if d[15] != 1 {
		return false
	}
	for i := 0; i < 15; i++ {
		if d[i] != 0 {
			return false
		}
	}
	return true
}

var (
	ipv4MappedIPv6Prefix = []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xff, 0xff}
)

func (d ipv6data) canonicalize() ipAddrData {
	if !bytes.Equal(d[:12], ipv4MappedIPv6Prefix) {
		return d
	}
	var canonicalized ipv4data
	copy(canonicalized[:], d[12:])
	return canonicalized
}

// IPAddress represents an IP (v4 or v6) address. In contrast to `net.IP`, it can be used as keys in maps.
type IPAddress struct {
	data ipAddrData
}

// Family returns the address family of this IP address.
func (a IPAddress) Family() Family {
	if a.data == nil {
		return InvalidFamily
	}
	return a.data.family()
}

// AsNetIP returns the `net.IP` representation of this IP address. If the IP address is invalid, `nil` is returned.
func (a IPAddress) AsNetIP() net.IP {
	if a.data == nil {
		return nil
	}
	return net.IP(a.data.bytes())
}

// String returns the string representation of this IP address.
func (a IPAddress) String() string {
	return a.AsNetIP().String()
}

// IsValid checks if the IP address is valid, i.e., is non-nil.
func (a IPAddress) IsValid() bool {
	return a.data != nil
}

// IsLoopback checks if the IP is a local loopback address (127.0.0.0/8 or ::1)
func (a IPAddress) IsLoopback() bool {
	return a.data != nil && a.data.isLoopback()
}

// IsUnspecified checks if the IP is the unspecified address representing all local IPs.
func (a IPAddress) IsUnspecified() bool {
	if a.data == nil {
		return false
	}
	for _, b := range a.data.bytes() {
		if b != 0 {
			return false
		}
	}
	return true
}

// IPFromBytes returns the IP address from the given byte slice. The byte slice must be of length 4 or 16, otherwise
// the invalid IP address is returned.
func IPFromBytes(data []byte) IPAddress {
	if len(data) == 4 {
		ipv4data := ipv4data{}
		copy(ipv4data[:], data)
		return IPAddress{data: ipv4data}
	}
	if len(data) != 16 {
		return IPAddress{}
	}
	ipv6data := ipv6data{}
	copy(ipv6data[:], data)
	return IPAddress{data: ipv6data.canonicalize()}
}

// FromNetIP converts a `net.IP` object to an `IPAddress`.
func FromNetIP(ip net.IP) IPAddress {
	return IPFromBytes(ip)
}

// ParseIP parses a string representation of an IP address.
func ParseIP(str string) IPAddress {
	return FromNetIP(net.ParseIP(str))
}
