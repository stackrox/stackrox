package netutil

import (
	"fmt"
	"strings"
)

// WithDefaultPort translates the given "<host>[:<port>]" string into a "<host>:<port>" string, adding the given
// default port if none is set. An empty string is always passed through unmodified.
func WithDefaultPort(hostAndMaybePort string, defaultPort uint16) string {
	if hostAndMaybePort == "" {
		return ""
	}
	if hostAndMaybePort[0] == '[' { // IPv6 ip+port notation
		if hostAndMaybePort[len(hostAndMaybePort)-1] != ']' {
			return hostAndMaybePort
		}
		return fmt.Sprintf("%s:%d", hostAndMaybePort, defaultPort)
	}
	switch strings.Count(hostAndMaybePort, ":") {
	case 0: // IPv4 or hostname
		return fmt.Sprintf("%s:%d", hostAndMaybePort, defaultPort)
	case 1: // IPv4 or hostname + port
		return hostAndMaybePort
	default: // >= 2 colons => IPv6, no port since doesn't start with '['
		return fmt.Sprintf("[%s]:%d", hostAndMaybePort, defaultPort)
	}
}
