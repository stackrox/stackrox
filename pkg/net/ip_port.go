package net

import (
	"fmt"
	"net"
	"strconv"
)

// IPPortPair is a purely numerical representation of an ip:port pair. It can be used as a map key.
type IPPortPair struct {
	Address IPAddress
	Port    uint16
}

// IsValid checks if the ip:port pair is valid.
func (e IPPortPair) IsValid() bool {
	return e.Address.IsValid()
}

// String returns a string representation of this ip:port pair.
func (e IPPortPair) String() string {
	if e.Port == 0 {
		return e.Address.String()
	}
	var ldelim, rdelim string
	if e.Address.Family() == IPv6 {
		ldelim, rdelim = "[", "]"
	}
	return fmt.Sprintf("%s%s%s:%d", ldelim, e.Address.String(), rdelim, e.Port)
}

// ParseIPPortPair parses a string representation of an ip:port pair. An invalid ip:port pair is returned if the string
// could not be parsed.
func ParseIPPortPair(str string) IPPortPair {
	host, portStr, err := net.SplitHostPort(str)
	if err != nil {
		return IPPortPair{
			Address: ParseIP(str),
		}
	}
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return IPPortPair{}
	}
	parsedIP := ParseIP(host)
	if !parsedIP.IsValid() {
		return IPPortPair{}
	}
	return IPPortPair{
		Address: parsedIP,
		Port:    uint16(port),
	}
}
