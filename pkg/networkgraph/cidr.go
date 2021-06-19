package networkgraph

import (
	"net"

	"github.com/pkg/errors"
)

// ValidateCIDR validates a given CIDR string semantics and whether it is supported by StackRox's network graph.
func ValidateCIDR(cidr string) (*net.IPNet, error) {
	if cidr == "" {
		return nil, errors.New("CIDR block is empty")
	}

	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	if ipNet.IP.IsLoopback() {
		return nil, errors.New("loopback address not supported")
	}

	if ipNet.IP.IsUnspecified() {
		return nil, errors.Errorf("unspecified address %s not supported", ipNet.IP)
	}
	return ipNet, nil
}
