package netutil

import "net"

// IsIPAddress returns whether the given host is a numeric IP address.
func IsIPAddress(host string) bool {
	return net.ParseIP(host) != nil
}
