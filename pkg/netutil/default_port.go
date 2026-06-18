package netutil

import (
	"net"
	"strconv"
	"strings"
)

// WithDefaultPort translates the given "<host>[:<port>]" string into a "<host>:<port>" string, adding the given
// default port if none is set. An empty string is always passed through unmodified.
func WithDefaultPort(hostAndMaybePort string, defaultPort uint16) string {
	if hostAndMaybePort == "" {
		return ""
	}
	port := strconv.FormatUint(uint64(defaultPort), 10)
	if hostAndMaybePort[0] == '[' { // IPv6 ip+port notation
		if hostAndMaybePort[len(hostAndMaybePort)-1] != ']' {
			return hostAndMaybePort
		}
		return net.JoinHostPort(hostAndMaybePort[1:len(hostAndMaybePort)-1], port)
	}
	switch strings.Count(hostAndMaybePort, ":") {
	case 0: // IPv4 or hostname
		return net.JoinHostPort(hostAndMaybePort, port)
	case 1: // IPv4 or hostname + port
		return hostAndMaybePort
	default: // >= 2 colons => IPv6, no port since doesn't start with '['
		return net.JoinHostPort(hostAndMaybePort, port)
	}
}
