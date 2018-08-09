package utils

import (
	"testing"

	"github.com/stackrox/rox/generated/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestNewImage(t *testing.T) {
	var cases = []struct {
		ImageString   string
		ExpectedImage *v1.Image
	}{
		{
			ImageString: "nginx:latest@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &v1.Image{
				Name: &v1.ImageName{
					Registry: "docker.io",
					Remote:   "library/nginx",
					Tag:      "latest",
					Sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
					FullName: "docker.io/library/nginx:latest",
				},
			},
		},
		{
			ImageString: "stackrox.io/prevent:1.0@sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
			ExpectedImage: &v1.Image{
				Name: &v1.ImageName{
					Registry: "stackrox.io",
					Remote:   "prevent",
					Tag:      "1.0",
					Sha:      "sha256:adea4f68096fded167603ba6663ed615a80e090da68eb3c9e2508c15c8368401",
					FullName: "stackrox.io/prevent:1.0",
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.ImageString, func(t *testing.T) {
			assert.Equal(t, c.ExpectedImage, GenerateImageFromString(c.ImageString))
		})
	}
}
