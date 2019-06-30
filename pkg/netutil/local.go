package netutil

import (
	"net"
	"strings"
)

// IsLocalHost checks whether the given hostname or IP address refers to the local host.
func IsLocalHost(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return strings.ToLower(host) == "localhost"
}

// IsLocalEndpoint checks whether the given endpoint refers to the local host. If the endpoint can't be parsed, false
// is returned.
func IsLocalEndpoint(endpoint string) bool {
	host, _, _, err := ParseEndpoint(endpoint)
	if err != nil {
		return false
	}
	return IsLocalHost(host)
}
