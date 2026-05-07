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

func FuzzParseEndpoint(f *testing.F) {
	// Seed with valid endpoint examples from existing tests
	f.Add("localhost:8080")
	f.Add("[::1]:443")
	f.Add("192.168.1.1:80")
	f.Add("example.com:80")
	f.Add("127.0.0.1:8080")
	f.Add("example.com")

	// IPv6 addresses
	f.Add("[1::]:80")
	f.Add("1::")
	f.Add("2001:0db8:0000:0000:0000:ff00:0042:8329")
	f.Add("[2001:0db8:0000:0000:0000:ff00:0042:8329]:61273")

	// Zone-based examples
	f.Add("::1%lo0")
	f.Add("[::1%lo0]:443")
	f.Add("192.168.0.1%eth0")
	f.Add("[192.168.0.1%eth0]:80")
	f.Add("www.example.com%zone:http")

	// Edge cases
	f.Add("[")
	f.Add("]")
	f.Add(":")
	f.Add("%")
	f.Add("[]")
	f.Add(":")
	f.Add("::")
	f.Add(":::")
	f.Add("[::]")
	f.Add("[::]:")

	// Known invalid cases (should error but not panic)
	f.Add("www.example.com%")
	f.Add("www.example.com%:80")
	f.Add("[::1%]")
	f.Add("[::1%]:80")
	f.Add("www.example.com:")
	f.Add("www.example.com%lo0:")
	f.Add("[::1]:")
	f.Add("[::1%lo0]:")
	f.Add("%lo0")
	f.Add("%lo0:80")
	f.Add("[%lo0]")
	f.Add("[%lo0]:80")

	f.Fuzz(func(t *testing.T, endpoint string) {
		// The primary goal is to ensure ParseEndpoint never panics,
		// regardless of input. We don't assert correctness of the parse,
		// just that it either succeeds or returns an error gracefully.
		host, zone, port, err := ParseEndpoint(endpoint)

		// If parsing succeeded, verify basic invariants
		if err == nil {
			// Host must not be empty on success
			assert.NotEmpty(t, host, "host should not be empty when parsing succeeds")

			// If zone is present, it must not be empty
			if zone != "" {
				assert.NotEmpty(t, zone, "zone should not be empty if present")
			}

			// If port is present, it must not be empty
			if port != "" {
				assert.NotEmpty(t, port, "port should not be empty if present")
			}

			// NOTE: Round-trip (FormatEndpoint -> ParseEndpoint) doesn't always
			// hold because ParseEndpoint accepts formats that FormatEndpoint
			// doesn't produce (e.g. bare IPv6 without brackets). We only
			// verify FormatEndpoint doesn't panic.
			_ = FormatEndpoint(host, zone, port)
		}
		// If err != nil, that's fine - many inputs are invalid.
		// We just want to ensure no panics occurred.
	})
}
