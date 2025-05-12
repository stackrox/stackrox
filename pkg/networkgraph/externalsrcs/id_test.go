package externalsrcs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewClusterScopedID(t *testing.T) {
	testCases := []struct {
		description, cluster, cidr, expected, expectedError string
	}{
		{
			description: "happy path",
			cluster:     "cluster1",
			cidr:        "10.0.0.0/24",
			expected:    "cluster1__MTAuMC4wLjAvMjQ",
		},
		{
			description: "cidr from IP",
			cluster:     "cluster1",
			cidr:        "1.1.1.1/30",
			expected:    "cluster1__MS4xLjEuMC8zMA",
		},
		{
			description: "cidr from network",
			cluster:     "cluster1",
			cidr:        "1.1.1.0/30",
			expected:    "cluster1__MS4xLjEuMC8zMA",
		},
		{
			description:   "wrong cidr",
			cluster:       "cluster1",
			cidr:          "1.1.1.0/99",
			expected:      "__",
			expectedError: "CIDR 1.1.1.0/99 is invalid",
		},
		{
			description:   "IP instead of CIDR",
			cluster:       "cluster1",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			description:   "wrong cidr and no cluster",
			cluster:       "",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			description:   "invalid cluster and invalid cidr",
			cluster:       "_",
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			description:   "invalid cluster and no cidr",
			cluster:       "_",
			cidr:          "",
			expected:      "__",
			expectedError: "CIDR must be provided",
		},
		{
			description:   "no cluster",
			cluster:       "",
			cidr:          "1.1.1.0/30",
			expected:      "__",
			expectedError: "cluster ID must be specified",
		},
		{
			description:   "invalid cluster",
			cluster:       "_",
			cidr:          "1.1.1.0/30",
			expected:      "__",
			expectedError: `cluster ID _ must not contain "_"`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
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

func TestNewGlobalScopedID(t *testing.T) {
	testCases := []struct {
		cidr, expected string
		expectedError  string
	}{
		{
			cidr:     "1.1.1.1/30",
			expected: "__MS4xLjEuMC8zMA",
		},
		{
			cidr:     "1.1.1.0/30",
			expected: "__MS4xLjEuMC8zMA",
		},
		{
			cidr:          "1.1.1.1",
			expected:      "__",
			expectedError: "CIDR 1.1.1.1 is invalid",
		},
		{
			cidr:          "",
			expected:      "__",
			expectedError: "CIDR must be provided",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.cidr+tc.expectedError, func(t *testing.T) {
			actual, err := NewGlobalScopedScopedID(tc.cidr)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.expected, actual.String())
		})
	}
}

func TestNetworkFromId(t *testing.T) {
	testCases := []struct {
		id, cidr, expectedError string
	}{
		{
			cidr: "1.1.1.0/30",
			id:   "__MS4xLjEuMC8zMA",
		},
		{
			cidr:          "",
			id:            "__",
			expectedError: `suffix part not found in ID "__"`,
		},
		{
			cidr:          "",
			id:            "__MS4xLjEuMC8zMA===",
			expectedError: `decoding suffix MS4xLjEuMC8zMA=== to CIDR: illegal base64 data at input byte 14`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.cidr+tc.expectedError, func(t *testing.T) {
			actual, err := NetworkFromID(tc.id)
			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
			assert.Equal(t, tc.cidr, actual)
		})
	}
}
