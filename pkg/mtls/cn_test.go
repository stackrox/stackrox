package mtls

import (
	"testing"

	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestSubject(t *testing.T) {
	cases := []struct {
		input    Subject
		cn       string
		hostname string
		ou       string
		name     cfcsr.Name
	}{
		{
			input:    Subject{ServiceType: storage.ServiceType_CENTRAL_SERVICE, Identifier: "Central"},
			cn:       "CENTRAL_SERVICE: Central",
			hostname: "central.stackrox",
			ou:       "CENTRAL_SERVICE",
			name: cfcsr.Name{
				OU: "CENTRAL_SERVICE",
			},
		},
		{
			input:    Subject{ServiceType: storage.ServiceType_SENSOR_SERVICE, Identifier: "1234"},
			cn:       "SENSOR_SERVICE: 1234",
			hostname: "sensor.stackrox",
			ou:       "SENSOR_SERVICE",
			name: cfcsr.Name{
				OU: "SENSOR_SERVICE",
			},
		},
		{
			input:    Subject{ServiceType: storage.ServiceType_COLLECTOR_SERVICE, Identifier: "456"},
			cn:       "COLLECTOR_SERVICE: 456",
			hostname: "collector.stackrox",
			ou:       "COLLECTOR_SERVICE",
			name: cfcsr.Name{
				OU: "COLLECTOR_SERVICE",
			},
		},
		{
			input:    Subject{ServiceType: storage.ServiceType_SCANNER_DB_SERVICE, Identifier: "456"},
			cn:       "SCANNER_DB_SERVICE: 456",
			hostname: "scanner-db.stackrox",
			ou:       "SCANNER_DB_SERVICE",
			name: cfcsr.Name{
				OU: "SCANNER_DB_SERVICE",
			},
		},
		{
			input:    Subject{ServiceType: storage.ServiceType_ADMISSION_CONTROL_SERVICE, Identifier: "456"},
			cn:       "ADMISSION_CONTROL_SERVICE: 456",
			hostname: "admission-control.stackrox",
			ou:       "ADMISSION_CONTROL_SERVICE",
			name: cfcsr.Name{
				OU: "ADMISSION_CONTROL_SERVICE",
			},
		},
		{
			input:    Subject{ServiceType: storage.ServiceType_CENTRAL_DB_SERVICE, Identifier: "Central DB/serialNumber=14079776202872467048"},
			cn:       "CENTRAL_DB_SERVICE: Central DB/serialNumber=14079776202872467048",
			hostname: "central-db.stackrox",
			ou:       "CENTRAL_DB_SERVICE",
			name: cfcsr.Name{
				OU: "CENTRAL_DB_SERVICE",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.cn, func(t *testing.T) {
			assert.Equal(t, c.cn, c.input.CN())
			assert.Equal(t, c.hostname, c.input.Hostname())
			assert.Equal(t, c.ou, c.input.OU())
			assert.Equal(t, c.name, c.input.Name())
		})
	}
}

func TestCommonNameFromString(t *testing.T) {
	cases := []struct {
		input    string
		expected Subject
	}{
		{
			input: "SENSOR_SERVICE: de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			expected: Subject{
				ServiceType: storage.ServiceType_SENSOR_SERVICE,
				Identifier:  "de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			},
		},
		{
			input: "Something Malformed",
			expected: Subject{
				ServiceType: storage.ServiceType_UNKNOWN_SERVICE,
				Identifier:  "Something Malformed",
			},
		},
		{
			input: "UNKNOWN_SOMETHING_OR_OTHER: de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			expected: Subject{
				ServiceType: storage.ServiceType_UNKNOWN_SERVICE,
				Identifier:  "de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := SubjectFromCommonName(c.input)
			assert.Equal(t, c.expected, got)
		})
	}
}
