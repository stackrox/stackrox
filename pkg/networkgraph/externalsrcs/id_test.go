package externalsrcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClusterScopedID(t *testing.T) {
	testCases := []struct {
		cluster, cidr, expected string
		expectedError           string
	}{
		{
			cluster:  "cluster1",
			cidr:     "10.0.0.0/24",
			expected: "cluster1__MTAuMC4wLjAvMjQ",
		},
		{
			cluster:  "cluster1",
			cidr:     "1.1.1.1/30",
			expected: "cluster1__MS4xLjEuMC8zMA",
		},
		{
			cluster:  "cluster1",
			cidr:     "1.1.1.0/30",
			expected: "cluster1__MS4xLjEuMC8zMA",
		},
		{
			cluster:       "cluster1",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			cluster:       "",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			cluster:       "_",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			cluster:       "_",
			cidr:          "",
			expected:      "__",
			expectedError: "CIDR must be provided",
		},
		{
			cluster:       "",
			cidr:          "1.1.1.0/30",
			expected:      "__",
			expectedError: "cluster ID must be specified",
		},
		{
			cluster:       "_",
			cidr:          "1.1.1.0/30",
			expected:      "__",
			expectedError: `cluster ID _ must not contain "_"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.expected+tc.expectedError, func(t *testing.T) {
			actual, err := NewClusterScopedID(tc.cluster, tc.cidr)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expected, actual.String())
		})
	}
}
