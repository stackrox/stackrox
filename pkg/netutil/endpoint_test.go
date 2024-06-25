package netutil

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type endpointTestCase struct {
	endpoint  string
	canonical bool
}

func makeEndpoints(host, zone, port string) []endpointTestCase {
	var result []endpointTestCase

	hostZone := host
	if zone != "" {
		hostZone += "%" + zone
	}

	if port == "" {
		result = append(result, endpointTestCase{endpoint: hostZone, canonical: true})
		result = append(result, endpointTestCase{endpoint: fmt.Sprintf("[%s]", hostZone)})
		return result
	}

	needsBrackets := strings.ContainsRune(hostZone, ':')
	if !needsBrackets {
		result = append(result, endpointTestCase{endpoint: fmt.Sprintf("%s:%s", hostZone, port), canonical: true})
	}
	result = append(result, endpointTestCase{endpoint: fmt.Sprintf("[%s]:%s", hostZone, port), canonical: needsBrackets})

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

				for _, epTC := range endpoints {
					ep := epTC.endpoint
					parsedHost, parsedZone, parsedPort, err := ParseEndpoint(ep)
					assert.NoError(t, err, "error parsing endpoint %s", ep)
					assert.Equal(t, host, parsedHost, "host mismatch for endpoint %s", ep)
					assert.Equal(t, zone, parsedZone, "zone mismatch for endpoint %s", ep)
					assert.Equal(t, port, parsedPort, "port mismatch for endpoint %s", ep)

					if epTC.canonical {
						formattedEndpoint := FormatEndpoint(parsedHost, parsedZone, parsedPort)
						assert.Equal(t, ep, formattedEndpoint, "FormatEndpoint did not result in original endpoint")
					}
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
