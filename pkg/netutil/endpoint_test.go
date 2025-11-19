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

func TestParseEndpoint(t *testing.T) {
	tcs := []struct {
		// endpoint must have no path and no scheme
		endpoint string
		wantHost string
		wantZone string
		wantPort string
		wantErr  bool
	}{
		{"example.com:80", "example.com", "", "80", false},
		{"127.0.0.1:8080", "127.0.0.1", "", "8080", false},
		{"example.com", "example.com", "", "", false},
		{"[1::]:80", "1::", "", "80", false},
		{"1::", "1::", "", "", false},
		{"2001:0db8:0000:0000:0000:ff00:0042:8329", "2001:0db8:0000:0000:0000:ff00:0042:8329", "", "", false},
		// This address with port is strictly conforming to RFC2732
		{"[2001:0db8:0000:0000:0000:ff00:0042:8329]:61273", "2001:0db8:0000:0000:0000:ff00:0042:8329", "", "61273", false},
		// This address with port is NOT strictly conforming to RFC2732. We do not support this for now
		// {"2001:0db8:0000:0000:0000:ff00:0042:8329:61273", "2001:0db8:0000:0000:0000:ff00:0042:8329", "", "61273", false},
	}
	for _, tc := range tcs {
		t.Run(tc.endpoint, func(t *testing.T) {
			gotHost, gotZone, gotPort, err := ParseEndpoint(tc.endpoint)
			assert.Equal(t, tc.wantHost, gotHost)
			assert.Equal(t, tc.wantZone, gotZone)
			assert.Equal(t, tc.wantPort, gotPort)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestParseEndpoint_Valid(t *testing.T) {

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
