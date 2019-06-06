package zip

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSafeFilename(t *testing.T) {

	var cases = []struct {
		input    string
		expected string
	}{
		{
			input:    "Openshift cluster",
			expected: "openshift-cluster",
		},
		{
			input:    "openshift cluster 1/2",
			expected: "openshift-cluster-1-2",
		},
		{
			input:    "  openshift   cluster   ",
			expected: "openshift-cluster",
		},
		{
			input:    "∆openshift cluster∆",
			expected: "openshift-cluster",
		},
		{
			input:    "openshift cluster 1///2",
			expected: "openshift-cluster-1-2",
		},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			assert.Equal(t, c.expected, GetSafeFilename(c.input))
		})
	}

}
