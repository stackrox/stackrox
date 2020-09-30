package renderer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComputeOverrides(t *testing.T) {
	cases := []struct {
		ref      string
		expected map[string]string
	}{
		{
			ref: "stackrox.io/main:1.2.3",
		},
		{
			ref: "stackrox.io/main:4.5.6",
			expected: map[string]string{
				"Tag": "4.5.6",
			},
		},
		{
			ref: "stackrox.io/sub-repo/main:1.2.3",
			expected: map[string]string{
				"Registry": "stackrox.io/sub-repo",
			},
		},
		{
			ref: "stackrox.io/sub-repo/main:4.5.6",
			expected: map[string]string{
				"Registry": "stackrox.io/sub-repo",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "stackrox.io/mymain:1.2.3",
			expected: map[string]string{
				"Name": "mymain",
			},
		},
		{
			ref: "stackrox.io/mymain:4.5.6",
			expected: map[string]string{
				"Name": "mymain",
				"Tag":  "4.5.6",
			},
		},
		{
			ref: "stackrox.io/sub-repo/mymain:4.5.6",
			expected: map[string]string{
				"Name": "sub-repo/mymain",
				"Tag":  "4.5.6",
			},
		},
		{
			ref: "docker.io/stackrox/main:1.2.3",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
			},
		},
		{
			ref: "docker.io/stackrox/main:4.5.6",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Tag":      "4.5.6",
			},
		},
		{
			ref: "docker.io/stackrox/mymain:1.2.3",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Name":     "mymain",
			},
		},
		{
			ref: "docker.io/stackrox/mymain:4.5.6",
			expected: map[string]string{
				"Registry": "docker.io/stackrox",
				"Name":     "mymain",
				"Tag":      "4.5.6",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.ref, func(t *testing.T) {
			overrides := computeImageOverrides(c.ref, "stackrox.io", "main", "1.2.3")
			assert.Equal(t, c.expected, overrides)
		})
	}
}
