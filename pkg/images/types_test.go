package images

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	image := &v1.Image{
		Registry: "docker.io",
		Remote:   "library/nginx",
		Tag:      "latest",
		Sha:      "adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
	}
	newImage := GenerateImageFromString("nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401")
	assert.Equal(t, image, newImage)
}
