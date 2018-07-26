package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDigest(t *testing.T) {
	cases := []struct {
		sha      string
		expected string
	}{
		{
			sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			expected: "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
		},
		{
			sha:      "",
			expected: "",
		},
	}
	for _, c := range cases {
		t.Run(c.sha, func(t *testing.T) {
			assert.Equal(t, c.expected, NewDigest(c.sha).Digest())
		})
	}
}
