package netutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeEndpoints(host, zone, port string) []string {
	var result []string

	hostZone := host
	if zone != "" {
		hostZone += "%" + zone
	}

	if port == "" {
		result = append(result, hostZone)
		result = append(result, fmt.Sprintf("[%s]", hostZone))
		return result
	}

	if !strings.ContainsRune(hostZone, ':') {
		result = append(result, fmt.Sprintf("%s:%s", hostZone, port))
	}
	result = append(result, fmt.Sprintf("[%s]:%s", hostZone, port))

	return result
}

func TestParseEndpoint_Valid(t *testing.T) {
	t.Parallel()

	hosts := []string{
		"192.168.0.1",
		"::1",
		"www.example.com",
	}
	zones := []string{
		"",
		"lo0",
	}
	ports := []string{
		"",
		"80",
		"http",
	}

	for _, host := range hosts {
		for _, zone := range zones {
			for _, port := range ports {
				endpoints := makeEndpoints(host, zone, port)

				for _, ep := range endpoints {
					parsedHost, parsedZone, parsedPort, err := ParseEndpoint(ep)
					assert.NoError(t, err, "error parsing endpoint %s", ep)
					assert.Equal(t, host, parsedHost, "host mismatch for endpoint %s", ep)
					assert.Equal(t, zone, parsedZone, "zone mismatch for endpoint %s", ep)
					assert.Equal(t, port, parsedPort, "port mismatch for endpoint %s", ep)
				}
			}
		}
	}
}

func TestParseEndpoint_Invalid(t *testing.T) {
	invalidEndpoints := []string{
		// empty zones
		"www.example.com%",
		"www.example.com%:80",
		"[::1%]",
		"[::1%]:80",
		// empty ports
		"www.example.com:",
		"www.example.com%lo0:",
		"[::1]:",
		"[::1%lo0]:",
		// empty host
		"%lo0",
		"%lo0:80",
		"[%lo0]",
		"[%lo0]:80",
	}

	for _, ep := range invalidEndpoints {
		_, _, _, err := ParseEndpoint(ep)
		assert.Error(t, err)
	}
}
