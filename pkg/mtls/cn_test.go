package mtls

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestCommonNameFromString(t *testing.T) {
	cases := []struct {
		input    string
		expected CommonName
	}{
		{
			input: "SENSOR_SERVICE: de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			expected: CommonName{
				ServiceType: v1.ServiceType_SENSOR_SERVICE,
				Identifier:  "de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			},
		},
		{
			input: "Something Malformed",
			expected: CommonName{
				ServiceType: v1.ServiceType_UNKNOWN_SERVICE,
				Identifier:  "Something Malformed",
			},
		},
		{
			input: "UNKNOWN_SOMETHING_OR_OTHER: de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			expected: CommonName{
				ServiceType: v1.ServiceType_UNKNOWN_SERVICE,
				Identifier:  "de23cc85-4fb0-4ba4-9092-771cb4f23b97",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := CommonNameFromString(c.input)
			assert.Equal(t, c.expected, got)
		})
	}
}
