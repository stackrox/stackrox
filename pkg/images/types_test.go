package images

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	image := &v1.Image{
		Name: &v1.ImageName{
			Registry: "docker.io",
			Remote:   "library/nginx",
			Tag:      "latest",
			Sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			FullName: "docker.io/library/nginx:latest",
		},
	}
	newImage := GenerateImageFromString("nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401")
	assert.Equal(t, image, newImage)
}

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
