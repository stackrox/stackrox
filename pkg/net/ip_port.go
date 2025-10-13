package net

import (
	"fmt"
	"net"
	"strconv"
)

// NetworkPeerID is a purely numerical representation of an ip:port pair. It can be used as a map key.
// `Address` and `IPNetwork` fields must be used mutually exclusively. `Address` is required to represent an IP address
// whereas `IPNetwork` is required to represent networks.
type NetworkPeerID struct {
	Address IPAddress
	Port    uint16

	IPNetwork IPNetwork
}

// IsAddressValid checks if the ip Address is valid.
// This does not check the validity of IPNetwork.
func (e NetworkPeerID) IsAddressValid() bool {
	return e.Address.IsValid()
}

// String returns a string representation of this ip:port pair.
func (e NetworkPeerID) String() string {
	addrPrefix := e.Address.String()
	isIPv6 := e.Address.Family() == IPv6
	if addrPrefix == "" {
		addrPrefix = e.IPNetwork.IP().String()
		isIPv6 = e.IPNetwork.IP().Family() == IPv6
		if e.IPNetwork.prefixLen > 0 {
			addrPrefix = fmt.Sprintf("%s/%d", addrPrefix, e.IPNetwork.prefixLen)
		}
	}
	if e.Port == 0 {
		return addrPrefix
	}
	var ldelim, rdelim string
	if isIPv6 {
		ldelim, rdelim = "[", "]"
	}
	return fmt.Sprintf("%s%s%s:%d", ldelim, addrPrefix, rdelim, e.Port)
}

// ParseIPPortPair parses a string representation of an ip:port pair. An invalid ip:port pair is returned if the string
// could not be parsed.
func ParseIPPortPair(str string) NetworkPeerID {
	host, portStr, err := net.SplitHostPort(str)
	if err != nil {
		return NetworkPeerID{
			Address: ParseIP(str),
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return NetworkPeerID{}
	}
	parsedIP := ParseIP(host)
	if !parsedIP.IsValid() {
		return NetworkPeerID{}
	}
	return NetworkPeerID{
		Address: parsedIP,
		Port:    uint16(port),
	}
}
