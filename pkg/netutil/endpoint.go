package netutil

import (
	"strings"

	"github.com/pkg/errors"
)

// ParseEndpoint parses an endpoint into a host, an optional zone, and an optional port. This is intended to be a more
// flexible replacement for `net.SplitHostPort` that works with or without a port specification.
func ParseEndpoint(endpoint string) (host, zone, port string, err error) {
	if endpoint == "" {
		err = errors.New("empty endpoint specified")
		return
	}

	var hostZone string
	if endpoint[0] == '[' {
		hostZoneEndIdx := strings.LastIndex(endpoint, "]")
		if hostZoneEndIdx == -1 {
			err = errors.New("missing ']' after opening bracket")
			return
		}
		hostZone = endpoint[1:hostZoneEndIdx]
		endpoint = endpoint[hostZoneEndIdx+1:]
	} else if strings.Count(endpoint, ":") > 1 { // more than two colons -> IPv6 address without port
		hostZone = endpoint
		endpoint = ""
	} else {
		hostZoneEndIdx := strings.IndexRune(endpoint, ':')
		if hostZoneEndIdx == -1 {
			hostZoneEndIdx = len(endpoint)
		}
		hostZone = endpoint[0:hostZoneEndIdx]
		endpoint = endpoint[hostZoneEndIdx:]
	}

	hostZoneComponents := strings.SplitN(hostZone, "%", 3)
	if len(hostZoneComponents) > 2 {
		err = errors.New("too many '%' characters in host/zone part")
		return
	}

	if endpoint != "" {
		if endpoint[0] != ':' {
			err = errors.New("expected ':' character or end of string after host/zone part")
			return
		}
		if len(endpoint) == 1 {
			err = errors.New("expected port name or number following ':' character")
			return
		}
		port = endpoint[1:]
	}

	host = hostZoneComponents[0]
	if host == "" {
		err = errors.New("empty host")
		return
	}
	if len(hostZoneComponents) > 1 {
		zone = hostZoneComponents[1]
		if zone == "" {
			err = errors.New("empty zone")
			return
		}
	}
	return
}
