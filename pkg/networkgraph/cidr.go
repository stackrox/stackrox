package networkgraph

import (
	"errors"
	"net"
)

// ValidateCIDR validates a given CIDR string semantics and whether it is supported by StackRox's network graph.
func ValidateCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "" {
		return nil, errors.New("CIDR block is empty")
	}

	ip, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	if ip.IsLoopback() {
		return nil, errors.New("loopback address not supported")
	}

	if ip.IsUnspecified() {
		return nil, errors.New("unspecified address not supported")
	}
	return ipNet, nil
}
