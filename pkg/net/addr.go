package net

import (
	"bytes"
	"net"

	"github.com/stackrox/rox/pkg/netutil"
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

// Bits returns the total bits in addresses represented by the family.
func (f Family) Bits() int {
	switch f {
	case IPv4:
		return 32
	case IPv6:
		return 128
	default:
		return 0
	}
}

type ipAddrData interface {
	family() Family
	bytes() []byte
	isLoopback() bool
	isPublic() bool
	canonicalize() ipAddrData
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

func (d ipv4data) isPublic() bool {
	netIP := net.IP(d.bytes())
	for _, privateIPNet := range netutil.IPV4PrivateNetworks {
		if privateIPNet.Contains(netIP) {
			return false
		}
	}
	return true
}

func (d ipv4data) canonicalize() ipAddrData {
	return d
}

type ipv6data [16]byte

func (d ipv6data) family() Family {
	return IPv6
}
func (d ipv6data) bytes() []byte {
	return d[:]
}
func (d ipv6data) isLoopback() bool {
	if netutil.IPV4MappedIPv6Loopback.Contains(net.IP(d.bytes())) {
		return true
	}

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

func (d ipv6data) isPublic() bool {
	netIP := net.IP(d.bytes())
	for _, privateIPNet := range netutil.IPV6PrivateNetworks {
		if privateIPNet.Contains(netIP) {
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

// IPAddressLess checks if the IP address a is less than the IP address b according to some defined ordering.
func IPAddressLess(a, b IPAddress) bool {
	aBytes, bBytes := a.data.bytes(), b.data.bytes()

	if len(aBytes) != len(bBytes) {
		return len(aBytes) < len(bBytes)
	}

	if a.data.family() != b.data.family() {
		return a.data.family() < b.data.family()
	}

	return bytes.Compare(aBytes, bBytes) < 0
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
	netIP := a.AsNetIP()
	if netIP == nil {
		return ""
	}
	return netIP.String()
}

// IsValid checks if the IP address is valid, i.e., is non-nil.
func (a IPAddress) IsValid() bool {
	return a.data != nil
}

// IsPublic checks if the IP is a public IP address (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16 for IPv4; fd00::/8 for
// IPv6). For an invalid IP, it returns false.
func (a IPAddress) IsPublic() bool {
	return a.data != nil && a.data.isPublic()
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

// IPNetwork represents an IP (v4 or v6) network.
type IPNetwork struct {
	ip        IPAddress
	prefixLen byte
}

// IP returns the network addres.
func (d IPNetwork) IP() IPAddress {
	return d.ip
}

// PrefixLen returns the length of IP network prefix.
func (d IPNetwork) PrefixLen() byte {
	return d.prefixLen
}

// Family returns the IP address family that the network belongs to.
func (d IPNetwork) Family() Family {
	return d.ip.Family()
}

// AsIPNet returns the IP address as `net.IPNet`.
func (d IPNetwork) AsIPNet() net.IPNet {
	if !d.IsValid() {
		return net.IPNet{}
	}

	return net.IPNet{
		IP:   d.ip.data.bytes(),
		Mask: net.CIDRMask(int(d.prefixLen), len(d.ip.data.bytes())*8),
	}
}

// IsValid returns true if this IPNetwork object is valid, else, returns false.
func (d IPNetwork) IsValid() bool {
	return d.ip.IsValid()
}

// Contains returns true if the IP network contains given ip.
func (d IPNetwork) Contains(ip IPAddress) bool {
	if !d.IsValid() {
		return false
	}

	ipNet := net.IPNet{
		IP:   d.ip.data.bytes(),
		Mask: net.CIDRMask(int(d.prefixLen), len(d.ip.data.bytes())*8),
	}
	return ipNet.Contains(ip.AsNetIP())
}

// String returns the IPNetwork in string form.
func (d IPNetwork) String() string {
	if !d.IsValid() {
		return ""
	}

	ipNet := &net.IPNet{
		IP:   d.ip.data.bytes(),
		Mask: net.CIDRMask(int(d.prefixLen), len(d.ip.data.bytes())*8),
	}
	return ipNet.String()
}

// IPNetworkFromCIDR converts a CIDR string string to an `IPNetwork`. In case of invalid string, an invalid IPNetwork is returned.
func IPNetworkFromCIDR(cidr string) IPNetwork {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return IPNetwork{}
	}

	ones, _ := ipNet.Mask.Size()

	return IPNetwork{
		ip:        IPFromBytes(ipNet.IP),
		prefixLen: byte(uint8(ones)),
	}
}

// IPNetworkFromIPNet converts a `net.IP` object to an `IPNetwork`. In case of invalid `ipNet`, an invalid IPNetwork is returned.
func IPNetworkFromIPNet(ipNet net.IPNet) IPNetwork {
	addr := IPFromBytes(ipNet.IP)
	ones, bits := ipNet.Mask.Size()
	if len(addr.data.bytes())*8 != bits {
		return IPNetwork{}
	}

	return IPNetwork{
		ip:        addr,
		prefixLen: byte(uint8(ones)),
	}
}

// IPNetworkFromCIDRBytes converts an IP network, in the form of array of bytes, to an `IPNetwork`. The array length must be 5 bytes
// for IpV4 and 17 bytes for IPV6, otherwise an invalid IPNetwork is returned.
func IPNetworkFromCIDRBytes(cidr []byte) IPNetwork {
	if len(cidr) != 5 && len(cidr) != 17 {
		return IPNetwork{}
	}

	n := len(cidr)
	return IPNetworkFromIPNet(net.IPNet{
		IP:   cidr[:n-1],
		Mask: net.CIDRMask(int(cidr[n-1]), (n-1)*8),
	})
}
